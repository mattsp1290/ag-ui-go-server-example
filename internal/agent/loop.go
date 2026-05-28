package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/runstore"
)

const maxIterations = 8

// defaultSystemPrompt is used when the request carries no system/developer
// message. The Codex Responses API requires non-empty instructions, and it also
// steers the model toward the read-only file_read tool.
const defaultSystemPrompt = "You are a helpful assistant operating in a read-only file workspace. " +
	"Use the file_read tool (with a workspace-relative path) to read file contents when the user asks about files. " +
	"You cannot create, modify, or delete files. Be concise."

// ensureSystemPrompt guarantees a leading system message so the Codex endpoint
// always receives instructions.
func ensureSystemPrompt(messages []*schema.Message) []*schema.Message {
	for _, m := range messages {
		if m.Role == schema.System {
			return messages
		}
	}
	return append([]*schema.Message{schema.SystemMessage(defaultSystemPrompt)}, messages...)
}

// Deps are the shared dependencies for running an agent turn.
type Deps struct {
	Model       model.ToolCallingChatModel // already tool-bound
	Tools       *Toolset
	Store       *runstore.Store
	AutoApprove bool
	Logger      *slog.Logger
}

// Run executes one AG-UI run: it streams the full event surface for either a
// fresh request or a resume of a previously interrupted run.
func Run(ctx context.Context, emit *Emitter, in *aguitypes.RunAgentInput, deps *Deps, threadID, runID string) {
	emit.RunStarted()

	key := runstore.Key(threadID, runID)
	var (
		st       *State
		messages []*schema.Message
	)

	// Resume path: rehydrate a paused run and settle the pending tool calls.
	if len(in.Resume) > 0 {
		if saved, ok := deps.Store.Load(key); ok {
			deps.Store.Delete(key)
			st = StateFromSnapshot(saved.State)
			messages = saved.Messages
			emit.StateSnapshot(st.Snapshot())

			approvals := approvalsFromResume(in.Resume)
			emit.StepStarted("tools")
			settlePendingToolCalls(ctx, emit, deps, saved.Pending, &messages, st, approvals)
			emit.StepFinished("tools")
		}
	}

	// Fresh path.
	if st == nil {
		st = NewState()
		st.Seed(in.State)
		messages = ensureSystemPrompt(toEinoMessages(in.Messages))
		emit.StateSnapshot(st.Snapshot())
	}

	for iter := 0; iter < maxIterations; iter++ {
		if emit.Err() != nil || ctx.Err() != nil {
			return // client disconnected
		}

		emit.StepStarted("llm")
		assistant, err := streamTurn(ctx, emit, deps.Model, messages)
		emit.StepFinished("llm")
		if err != nil {
			emit.RunError(fmt.Sprintf("model turn failed: %v", err))
			return
		}
		messages = append(messages, assistant)

		if len(assistant.ToolCalls) == 0 {
			break // final answer
		}

		emit.StepStarted("tools")
		for _, tc := range assistant.ToolCalls {
			emitToolProposal(emit, tc)
		}

		if !deps.AutoApprove {
			// Human-in-the-loop: pause for approval and finish with an interrupt.
			interrupts := make([]aguitypes.Interrupt, 0, len(assistant.ToolCalls))
			for _, tc := range assistant.ToolCalls {
				emit.ActivitySnapshot(aguievents.GenerateMessageID(), "approval_request",
					map[string]any{"text": fmt.Sprintf("Agent wants to call %s with %s — approve?", tc.Function.Name, tc.Function.Arguments)})
				interrupts = append(interrupts, aguitypes.Interrupt{
					ID:         tc.ID,
					Reason:     "tool_call",
					Message:    fmt.Sprintf("Approve %s(%s)?", tc.Function.Name, tc.Function.Arguments),
					ToolCallID: tc.ID,
					ResponseSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"approved": map[string]any{"type": "boolean"}},
						"required":   []any{"approved"},
					},
				})
			}
			deps.Store.Save(key, &runstore.Saved{
				Messages: messages,
				Pending:  assistant.ToolCalls,
				State:    st.Snapshot(),
			})
			emit.StateDelta(st.SetStatus("awaiting_approval"))
			emit.StepFinished("tools")
			emit.MessagesSnapshot(toAGUIMessages(messages))
			emit.RunFinishedInterrupt(interrupts)
			return
		}

		// Auto-approve: execute immediately and continue the loop.
		settlePendingToolCalls(ctx, emit, deps, assistant.ToolCalls, &messages, st, nil)
		emit.StepFinished("tools")
	}

	emit.StateDelta(st.SetStatus("done"))
	emit.Custom("agent_complete", map[string]any{"toolCalls": st.ToolCalls, "filesRead": st.FilesRead})
	emit.MessagesSnapshot(toAGUIMessages(messages))
	emit.RunFinishedSuccess()
}

// streamTurn streams one model turn, emitting reasoning and text events as
// chunks arrive, and returns the merged assistant message (Extra preserved so
// the codex model's reasoning items thread across turns).
func streamTurn(ctx context.Context, emit *Emitter, cm model.ToolCallingChatModel, messages []*schema.Message) (*schema.Message, error) {
	sr, err := cm.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	defer sr.Close()

	var chunks []*schema.Message
	textID := aguievents.GenerateMessageID()
	reasoningID := aguievents.GenerateMessageID()
	textOpen, reasoningOpen := false, false

	closeReasoning := func() {
		if reasoningOpen {
			emit.ReasoningMessageEnd(reasoningID)
			emit.ReasoningEnd(reasoningID)
			reasoningOpen = false
		}
	}

	for {
		chunk, recvErr := sr.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return nil, recvErr
		}
		if chunk.ReasoningContent != "" {
			if !reasoningOpen {
				emit.ReasoningStart(reasoningID)
				emit.ReasoningMessageStart(reasoningID)
				reasoningOpen = true
			}
			emit.ReasoningContent(reasoningID, chunk.ReasoningContent)
		}
		if chunk.Content != "" {
			closeReasoning() // reasoning precedes the visible answer
			if !textOpen {
				emit.TextStart(textID)
				textOpen = true
			}
			emit.TextContent(textID, chunk.Content)
		}
		chunks = append(chunks, chunk)
	}
	closeReasoning()
	if textOpen {
		emit.TextEnd(textID)
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("empty model stream")
	}
	return schema.ConcatMessages(chunks)
}

// emitToolProposal surfaces a proposed tool call (start/args/end), independent of
// whether it will be executed now or after an approval interrupt.
func emitToolProposal(emit *Emitter, tc schema.ToolCall) {
	emit.ToolStart(tc.ID, tc.Function.Name)
	emit.ToolArgs(tc.ID, tc.Function.Arguments)
	emit.ToolEnd(tc.ID)
}

// settlePendingToolCalls executes (or, when denied, records a denial for) each
// tool call, emitting the result and threading a role=tool message back into the
// conversation. A nil approvals map means "approve all" (auto-approve path).
func settlePendingToolCalls(ctx context.Context, emit *Emitter, deps *Deps, calls []schema.ToolCall, messages *[]*schema.Message, st *State, approvals map[string]bool) {
	for _, tc := range calls {
		approved := approvals == nil || approvals[tc.ID]
		var result string
		if approved {
			emit.ActivitySnapshot(aguievents.GenerateMessageID(), "tool_use",
				map[string]any{"text": fmt.Sprintf("Running %s(%s)", tc.Function.Name, tc.Function.Arguments)})
			out, err := deps.Tools.Run(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				out = fmt.Sprintf(`{"error":%q}`, err.Error())
			}
			result = out
			emit.StateDelta(st.RecordFileRead(extractPath(tc.Function.Arguments)))
		} else {
			result = `{"denied":true,"reason":"user did not approve this tool call"}`
		}
		emit.ToolResult(aguievents.GenerateMessageID(), tc.ID, result)
		*messages = append(*messages, schema.ToolMessage(result, tc.ID))
	}
}

// approvalsFromResume maps resume entries to per-tool-call approval. An entry is
// approved when status is "resolved" and its payload does not carry approved:false.
func approvalsFromResume(entries []aguitypes.ResumeEntry) map[string]bool {
	approvals := make(map[string]bool, len(entries))
	for _, e := range entries {
		approved := e.Status == aguitypes.ResumeStatusResolved
		if approved {
			if m, ok := e.Payload.(map[string]any); ok {
				if v, ok := m["approved"].(bool); ok {
					approved = v
				}
			}
		}
		approvals[e.InterruptID] = approved
	}
	return approvals
}

func extractPath(argsJSON string) string {
	var a struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &a)
	if a.Path == "" {
		return "(unknown)"
	}
	return a.Path
}
