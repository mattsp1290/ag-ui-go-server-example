package reasoning

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/agent"
)

// TestReasoningHandlerEmitsExpectedEvents drives the same scripted sequence
// that Handler() runs inside c.SendStreamWriter, asserting every required
// AG-UI event type appears in the output.
func TestReasoningHandlerEmitsExpectedEvents(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	emit := agent.NewEmitter(context.Background(), w, sse.NewSSEWriter(), "t1", "r1", nil)

	emit.RunStarted()

	reasoningID := aguievents.GenerateMessageID()
	emit.ReasoningStart(reasoningID)
	emit.ReasoningMessageStart(reasoningID)
	emit.ReasoningContent(reasoningID, "Let me think through this step by step...")
	emit.ReasoningMessageEnd(reasoningID)
	emit.ReasoningEnd(reasoningID)

	answerID := aguievents.GenerateMessageID()
	emit.TextStart(answerID)
	emit.TextContent(answerID, "Here is my response based on my reasoning.")
	emit.TextEnd(answerID)

	emit.MessagesSnapshot([]aguitypes.Message{})
	emit.RunFinishedSuccess()

	_ = w.Flush()
	out := buf.String()

	for _, want := range []string{
		`"type":"RUN_STARTED"`,
		`"type":"REASONING_START"`,
		`"type":"REASONING_MESSAGE_START"`,
		`"type":"REASONING_MESSAGE_CONTENT"`,
		`"type":"REASONING_MESSAGE_END"`,
		`"type":"REASONING_END"`,
		`"type":"TEXT_MESSAGE_START"`,
		`"type":"TEXT_MESSAGE_CONTENT"`,
		`"type":"TEXT_MESSAGE_END"`,
		`"type":"MESSAGES_SNAPSHOT"`,
		`"type":"RUN_FINISHED"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing event %s in output", want)
		}
	}

	// Reasoning block must close before text starts.
	if strings.Index(out, `"type":"REASONING_END"`) > strings.Index(out, `"type":"TEXT_MESSAGE_START"`) {
		t.Error("REASONING_END must appear before TEXT_MESSAGE_START")
	}
}
