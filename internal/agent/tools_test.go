package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/mattsp1290/eino-tools/fileops"
)

// TestToolsetReadOnly is the guard for the core safety property of this server:
// the agent may read files but must never have a tool that mutates the
// filesystem. If someone registers a write/edit/shell tool, this fails.
func TestToolsetReadOnly(t *testing.T) {
	ts, err := NewReadOnlyToolset(t.TempDir())
	if err != nil {
		t.Fatalf("NewReadOnlyToolset: %v", err)
	}

	infos := ts.Infos()
	if len(infos) != 1 {
		t.Fatalf("expected exactly 1 tool, got %d", len(infos))
	}
	if infos[0].Name != fileops.NameRead {
		t.Fatalf("expected only %q, got %q", fileops.NameRead, infos[0].Name)
	}

	// Any non-read tool must be rejected by the dispatcher.
	for _, name := range []string{fileops.NameWrite, fileops.NameEdit, "shell", "tracker_write"} {
		if _, err := ts.Run(context.Background(), name, `{}`); err == nil {
			t.Fatalf("tool %q should not be runnable on a read-only agent", name)
		} else if !strings.Contains(err.Error(), "only allows file_read") {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
	}
}
