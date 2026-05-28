package agent

import (
	"bufio"
	"context"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
)

// Emitter serializes AG-UI events to an SSE stream. It records the first write
// error and becomes a no-op afterward, so loop code can stay terse and check
// Err() at convenient points (a write error means the client disconnected).
//
// On the first write failure it invokes cancel (if set), which cancels the run
// context so an in-flight model stream aborts promptly instead of generating
// against a gone client. This is how client disconnect is detected: fasthttp's
// RequestCtx does not signal disconnect, only a failed SSE write does.
type Emitter struct {
	ctx      context.Context
	w        *bufio.Writer
	sse      *sse.SSEWriter
	threadID string
	runID    string
	cancel   context.CancelFunc
	err      error
}

// NewEmitter builds an Emitter bound to a request's SSE writer. cancel may be
// nil; when non-nil it is called once, on the first write error.
func NewEmitter(ctx context.Context, w *bufio.Writer, sw *sse.SSEWriter, threadID, runID string, cancel context.CancelFunc) *Emitter {
	return &Emitter{ctx: ctx, w: w, sse: sw, threadID: threadID, runID: runID, cancel: cancel}
}

// Err returns the first write error, if any.
func (e *Emitter) Err() error { return e.err }

func (e *Emitter) write(ev events.Event) {
	if e.err != nil {
		return
	}
	if err := e.sse.WriteEvent(e.ctx, e.w, ev); err != nil {
		e.err = err
		if e.cancel != nil {
			e.cancel() // abort the in-flight model stream on client disconnect
		}
	}
}

// --- run lifecycle ---

func (e *Emitter) RunStarted() { e.write(events.NewRunStartedEvent(e.threadID, e.runID)) }

func (e *Emitter) RunFinishedSuccess() {
	e.write(events.NewRunFinishedEventWithOptions(e.threadID, e.runID, events.WithSuccessOutcome()))
}

func (e *Emitter) RunFinishedInterrupt(interrupts []types.Interrupt) {
	e.write(events.NewRunFinishedEventWithOptions(e.threadID, e.runID, events.WithInterruptOutcome(interrupts)))
}

func (e *Emitter) RunError(msg string) {
	e.write(events.NewRunErrorEvent(msg, events.WithRunID(e.runID)))
}

// --- steps ---

func (e *Emitter) StepStarted(name string)  { e.write(events.NewStepStartedEvent(name)) }
func (e *Emitter) StepFinished(name string) { e.write(events.NewStepFinishedEvent(name)) }

// --- text messages ---

func (e *Emitter) TextStart(id string) {
	e.write(events.NewTextMessageStartEvent(id, events.WithRole("assistant")))
}

func (e *Emitter) TextContent(id, delta string) {
	if delta == "" {
		return // SDK rejects empty deltas
	}
	e.write(events.NewTextMessageContentEvent(id, delta))
}

func (e *Emitter) TextEnd(id string) { e.write(events.NewTextMessageEndEvent(id)) }

// --- reasoning ---

func (e *Emitter) ReasoningStart(id string) { e.write(events.NewReasoningStartEvent(id)) }

func (e *Emitter) ReasoningMessageStart(id string) {
	e.write(events.NewReasoningMessageStartEvent(id, "assistant"))
}

func (e *Emitter) ReasoningContent(id, delta string) {
	if delta == "" {
		return
	}
	e.write(events.NewReasoningMessageContentEvent(id, delta))
}

func (e *Emitter) ReasoningMessageEnd(id string) { e.write(events.NewReasoningMessageEndEvent(id)) }
func (e *Emitter) ReasoningEnd(id string)        { e.write(events.NewReasoningEndEvent(id)) }

// --- tool calls ---

func (e *Emitter) ToolStart(toolCallID, name string) {
	e.write(events.NewToolCallStartEvent(toolCallID, name))
}

func (e *Emitter) ToolArgs(toolCallID, delta string) {
	if delta == "" {
		return
	}
	e.write(events.NewToolCallArgsEvent(toolCallID, delta))
}

func (e *Emitter) ToolEnd(toolCallID string) { e.write(events.NewToolCallEndEvent(toolCallID)) }

func (e *Emitter) ToolResult(messageID, toolCallID, content string) {
	if content == "" {
		content = "(empty)"
	}
	e.write(events.NewToolCallResultEvent(messageID, toolCallID, content))
}

// --- state ---

func (e *Emitter) StateSnapshot(snapshot any) {
	e.write(events.NewStateSnapshotEvent(snapshot))
}

func (e *Emitter) StateDelta(ops []events.JSONPatchOperation) {
	if len(ops) == 0 {
		return
	}
	e.write(events.NewStateDeltaEvent(ops))
}

func (e *Emitter) MessagesSnapshot(msgs []types.Message) {
	e.write(events.NewMessagesSnapshotEvent(msgs))
}

// --- activity / custom ---

func (e *Emitter) ActivitySnapshot(messageID, activityType string, content any) {
	e.write(events.NewActivitySnapshotEvent(messageID, activityType, content))
}

func (e *Emitter) Custom(name string, value any) {
	e.write(events.NewCustomEvent(name, events.WithValue(value)))
}
