package agent

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

// PredictiveState runs the /predictive_state_updates flow (request 05): it shows
// the document filling in optimistically WHILE the agent generates it, then settles
// to the committed value.
//
// Signal convention (documented for the client): every STATE_DELTA whose path is
// under "/_predictive" is a PREDICTION — render it ghosted; every other delta is
// COMMITTED. As the model streams the new recipe steps, the growing draft is
// emitted as predictive deltas at "/_predictive/draft". On completion the final
// value is committed at "/recipe/steps" and the "/_predictive" namespace is
// removed. A client that drops every "/_predictive" delta still ends in the correct
// committed state (predictions are an enhancement, not the source of truth).
type PredictiveState struct {
	Deps *Deps
}

const predictiveSystemPrompt = "You are a recipe assistant. When the user asks you to write or " +
	"revise the recipe's steps, respond with the new steps only — one step per line, no " +
	"numbering or preamble. Each line is one step."

// predictiveDraftPath is the namespace marking a delta as a (ghosted) prediction.
const predictiveDraftPath = "/_predictive"

func (p PredictiveState) Run(ctx context.Context, emit *Emitter, in *aguitypes.RunAgentInput, threadID, runID string) {
	emit.RunStarted()

	doc := seedRecipe(in.State)
	emit.StateSnapshot(doc.Snapshot())

	messages := ensureSystemPrompt(toEinoMessages(in.Messages, p.Deps.Provider), predictiveSystemPrompt)

	emit.StepStarted("llm")
	full, ok := p.streamPredictive(ctx, emit, doc, messages)
	emit.StepFinished("llm")
	if !ok {
		return // streamPredictive already emitted RUN_ERROR or the client is gone
	}

	// Settle: commit the finalized steps to the real path, then clear the ghost.
	// The committed value is computed from the final generation, not "promoted"
	// from the last prediction, so a dropped/garbled prediction can't corrupt it.
	steps := splitSteps(full)
	if len(steps) > 0 {
		commit := []events.JSONPatchOperation{{Op: "replace", Path: "/recipe/steps", Value: steps}}
		if err := doc.Apply(commit); err == nil {
			emit.StateDelta(commit) // committed (not under /_predictive)
		}
	}
	// Remove the draft namespace (itself a predictive op — clients that ignore
	// predictions never created it and simply ignore this too).
	clear := []events.JSONPatchOperation{{Op: "remove", Path: predictiveDraftPath}}
	if err := doc.Apply(clear); err == nil {
		emit.StateDelta(clear)
	}

	// Short committed-state narration.
	msgID := events.GenerateMessageID()
	emit.TextStart(msgID)
	emit.TextContent(msgID, "Updated the recipe steps.")
	emit.TextEnd(msgID)

	emit.MessagesSnapshot([]aguitypes.Message{
		{ID: msgID, Role: aguitypes.RoleAssistant, Content: "Updated the recipe steps."},
	})
	emit.RunFinishedSuccess()
}

// streamPredictive streams the model turn, emitting the growing draft as predictive
// STATE_DELTAs under /_predictive/draft. It returns the full generated text and ok
// = true on success; on a model error it emits RUN_ERROR and returns ok = false.
func (p PredictiveState) streamPredictive(ctx context.Context, emit *Emitter, doc *DocState, messages []*schema.Message) (string, bool) {
	sr, err := p.Deps.BaseModel.Stream(ctx, messages)
	if err != nil {
		if errors.Is(err, context.Canceled) || emit.Err() != nil {
			return "", false
		}
		emit.RunError("the agent failed to generate a response")
		return "", false
	}
	defer sr.Close()

	var b strings.Builder
	for {
		if ctx.Err() != nil || emit.Err() != nil {
			return "", false
		}
		chunk, recvErr := sr.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			if errors.Is(recvErr, context.Canceled) || emit.Err() != nil {
				return "", false
			}
			emit.RunError("the agent failed to generate a response")
			return "", false
		}
		if chunk.Content == "" {
			continue
		}
		b.WriteString(chunk.Content)
		// Emit the current draft as a prediction. `add` to /_predictive is valid
		// whether or not it exists yet (it replaces the whole object), so each
		// update is a single self-contained op carrying the latest draft text.
		op := []events.JSONPatchOperation{{
			Op: "add", Path: predictiveDraftPath, Value: map[string]any{"draft": b.String()},
		}}
		if err := doc.Apply(op); err == nil {
			emit.StateDelta(op)
		}
	}
	return b.String(), true
}

// splitSteps turns the model's newline-separated step text into a steps array,
// trimming blank lines and any leading list markers.
func splitSteps(text string) []any {
	var steps []any
	for _, line := range strings.Split(text, "\n") {
		s := strings.TrimSpace(line)
		s = strings.TrimLeft(s, "-*0123456789. \t")
		if s != "" {
			steps = append(steps, s)
		}
	}
	return steps
}
