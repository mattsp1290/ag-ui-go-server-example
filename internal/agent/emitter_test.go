package agent

import (
	"bufio"
	"context"
	"errors"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
)

// failWriter always fails, forcing the SDK writer's write/flush error path.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("socket gone") }

// TestIsTransportErrorMatchesSDKWrappers pins the SDK's write/flush error wrapper
// strings that isTransportError matches. Disconnect detection (and the run
// cancellation it drives) depends on those exact substrings; if a future SDK bump
// rewords them, this fails loudly instead of silently reclassifying a client
// disconnect as an encoding error — which would leave a gone-client run generating
// (and billing) tokens until it finishes on its own.
func TestIsTransportErrorMatchesSDKWrappers(t *testing.T) {
	sw := sse.NewSSEWriter()
	w := bufio.NewWriter(failWriter{})

	// A valid event encodes fine; the failure happens at the socket write/flush.
	err := sw.WriteEvent(context.Background(), w, events.NewRunStartedEvent("t", "r"))
	if err == nil {
		t.Fatal("expected a write/flush error from the failing writer")
	}
	if !isTransportError(err) {
		t.Fatalf("isTransportError must classify an SDK write/flush failure as a transport error; "+
			"the SDK wrapper strings may have changed: %v", err)
	}
}
