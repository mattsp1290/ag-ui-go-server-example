package agent

import (
	"testing"

	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

func TestToAGUIMessagesIncludesReasoning(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.Assistant, Content: "answer", ReasoningContent: "let me think"},
	}
	out := toAGUIMessages(msgs)
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
	if out[0].Role != aguitypes.RoleAssistant {
		t.Errorf("out[0].Role = %q, want %q", out[0].Role, aguitypes.RoleAssistant)
	}
	if out[1].Role != aguitypes.RoleReasoning {
		t.Errorf("out[1].Role = %q, want %q", out[1].Role, aguitypes.RoleReasoning)
	}
	if out[1].Content != "let me think" {
		t.Errorf("out[1].Content = %q, want %q", out[1].Content, "let me think")
	}
}

func TestToAGUIMessagesNoExtraMessageWhenReasoningEmpty(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.Assistant, Content: "answer"},
	}
	out := toAGUIMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
}
