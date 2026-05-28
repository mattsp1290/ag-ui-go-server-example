package agent

import (
	"strings"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

// toEinoMessages maps the AG-UI request message history into eino messages.
// Roles eino has no use for here (reasoning/activity) are skipped.
func toEinoMessages(in []aguitypes.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(in))
	for _, m := range in {
		switch m.Role {
		case aguitypes.RoleUser:
			if text := messageText(m); text != "" {
				out = append(out, schema.UserMessage(text))
			}
		case aguitypes.RoleSystem, aguitypes.RoleDeveloper:
			if text := messageText(m); text != "" {
				out = append(out, schema.SystemMessage(text))
			}
		case aguitypes.RoleAssistant:
			content, _ := m.ContentString()
			out = append(out, &schema.Message{
				Role:      schema.Assistant,
				Content:   content,
				ToolCalls: toEinoToolCalls(m.ToolCalls),
			})
		case aguitypes.RoleTool:
			content, _ := m.ContentString()
			out = append(out, schema.ToolMessage(content, m.ToolCallID))
		}
	}
	return out
}

// messageText returns the message's text. ContentString applies to plain string
// content; for multimodal/structured content ([]InputContent) it returns ok=false,
// in which case we join the text fragments. Non-text fragments (image/audio/binary)
// are dropped — this agent has no vision/audio path — but the user's typed text is
// preserved instead of the whole turn being silently discarded. An empty result
// means the turn has no usable text; the caller skips it rather than inject a blank
// turn that would erase the user's actual input.
func messageText(m aguitypes.Message) string {
	if s, ok := m.ContentString(); ok {
		return s
	}
	parts, ok := m.ContentInputContents()
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Type == aguitypes.InputContentTypeText && p.Text != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func toEinoToolCalls(tcs []aguitypes.ToolCall) []schema.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]schema.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, schema.ToolCall{
			ID:       tc.ID,
			Type:     "function",
			Function: schema.FunctionCall{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
		})
	}
	return out
}

// toAGUIMessages converts the eino conversation into AG-UI messages for a
// MESSAGES_SNAPSHOT event, assigning fresh ids and conforming to the SDK's
// per-role validation rules.
func toAGUIMessages(msgs []*schema.Message) []aguitypes.Message {
	out := make([]aguitypes.Message, 0, len(msgs))
	for _, m := range msgs {
		am := aguitypes.Message{ID: aguievents.GenerateMessageID(), Content: m.Content}
		switch m.Role {
		case schema.System:
			am.Role = aguitypes.RoleSystem
		case schema.User:
			am.Role = aguitypes.RoleUser
		case schema.Assistant:
			am.Role = aguitypes.RoleAssistant
			am.ToolCalls = toAGUIToolCalls(m.ToolCalls)
		case schema.Tool:
			am.Role = aguitypes.RoleTool
			am.ToolCallID = m.ToolCallID
		default:
			continue
		}
		out = append(out, am)
	}
	return out
}

func toAGUIToolCalls(tcs []schema.ToolCall) []aguitypes.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]aguitypes.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, aguitypes.ToolCall{
			ID:       tc.ID,
			Type:     aguitypes.ToolCallTypeFunction,
			Function: aguitypes.FunctionCall{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
		})
	}
	return out
}
