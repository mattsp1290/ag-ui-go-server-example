package agent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
)

// failWriter always fails, forcing the SDK writer's write/flush error path.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("socket gone") }

// TestMessagesSnapshotScrubsEncryptedValues verifies that encrypted reasoning
// blobs do not appear in client-facing snapshots.
func TestMessagesSnapshotScrubsEncryptedValues(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	em := NewEmitter(context.Background(), w, sse.NewSSEWriter(), "t1", "r1", nil)
	withCipher := types.Message{
		ID:               "2",
		Role:             types.RoleAssistant,
		Content:          "thinking…",
		EncryptedValue:   "secret-ev",
		EncryptedContent: "secret-ec",
	}

	em.MessagesSnapshot([]types.Message{{ID: "1", Role: types.RoleUser, Content: "hello"}, withCipher})
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "secret-ev") || strings.Contains(out, "secret-ec") {
		t.Fatalf("snapshot leaked encrypted content:\n%s", out)
	}
	if !strings.Contains(out, "thinking…") {
		t.Fatalf("snapshot lost ordinary content:\n%s", out)
	}
}

// TestActivityDeltaAndReasoningEncryptedValue confirm the new emitter wrappers
// produce events with the correct type fields. We use a success writer (NopWriter)
// so we are testing serialization, not transport.
func TestActivityDeltaAndReasoningEncryptedValue(t *testing.T) {
	sw := sse.NewSSEWriter()
	w := bufio.NewWriter(&nopWriter{})
	em := NewEmitter(context.Background(), w, sw, "t1", "r1", nil)

	// ActivityDelta — verify it does not error (validates correctly).
	patch := []events.JSONPatchOperation{{Op: "replace", Path: "/content", Value: "updated"}}
	em.ActivityDelta("msg-1", "tool_use", patch)
	if em.EncErr() != nil {
		t.Errorf("ActivityDelta encoding error: %v", em.EncErr())
	}

	// ReasoningEncryptedValue — verify typed subtype constant is accepted.
	em.ReasoningEncryptedValue(events.ReasoningEncryptedValueSubtypeMessage, "msg-2", "cipher-blob")
	if em.EncErr() != nil {
		t.Errorf("ReasoningEncryptedValue encoding error: %v", em.EncErr())
	}
}

// nopWriter discards all bytes (stands in for a connected SSE client).
type nopWriter struct{}

func (*nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestIsTransportErrorMatchesSDKWrappers verifies that the shared emitter treats
// SDK write/flush failures as transport errors and cancels the run context.
func TestIsTransportErrorMatchesSDKWrappers(t *testing.T) {
	sw := sse.NewSSEWriter()
	w := bufio.NewWriter(failWriter{})

	ctx, cancel := context.WithCancel(context.Background())
	em := NewEmitter(ctx, w, sw, "t", "r", cancel)

	em.RunStarted()
	if em.Err() == nil {
		t.Fatal("expected emitter transport error from the failing writer")
	}
	if ctx.Err() == nil {
		t.Fatal("expected emitter to cancel the run context after transport failure")
	}
}
