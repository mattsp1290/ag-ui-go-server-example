package agent

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// fakeModel yields a fixed set of streamed chunks. It exists to exercise the
// REASONING_* emit path, which the live codex proof does not trigger (gpt-5.5
// returns encrypted reasoning, not plaintext ReasoningContent).
type fakeModel struct{ chunks []*schema.Message }

func (f *fakeModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.ConcatMessages(f.chunks)
}

func (f *fakeModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](len(f.chunks) + 1)
	go func() {
		defer sw.Close()
		for _, c := range f.chunks {
			sw.Send(c, nil)
		}
	}()
	return sr, nil
}

func (f *fakeModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) { return f, nil }

func TestStreamTurnEmitsReasoningThenText(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	emit := NewEmitter(context.Background(), w, sse.NewSSEWriter(), "t", "r", nil)

	fm := &fakeModel{chunks: []*schema.Message{
		{Role: schema.Assistant, ReasoningContent: "let me think"},
		{Role: schema.Assistant, Content: "Hello"},
		{Role: schema.Assistant, Content: " world"},
	}}

	msg, err := streamTurn(context.Background(), emit, fm, nil)
	if err != nil {
		t.Fatalf("streamTurn: %v", err)
	}
	_ = w.Flush()
	out := buf.String()

	for _, want := range []string{
		`"type":"REASONING_START"`,
		`"type":"REASONING_MESSAGE_START"`,
		`"type":"REASONING_MESSAGE_CONTENT"`,
		`"type":"REASONING_MESSAGE_END"`,
		`"type":"REASONING_END"`,
		`"type":"TEXT_MESSAGE_START"`,
		`"type":"TEXT_MESSAGE_CONTENT"`,
		`"type":"TEXT_MESSAGE_END"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing event %s", want)
		}
	}
	if strings.Index(out, `"type":"REASONING_END"`) > strings.Index(out, `"type":"TEXT_MESSAGE_START"`) {
		t.Errorf("reasoning block must close before text starts")
	}
	if msg.Content != "Hello world" {
		t.Errorf("merged content = %q, want %q", msg.Content, "Hello world")
	}
}
