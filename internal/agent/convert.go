package agent

import (
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

// toEinoMessages maps the AG-UI request message history into eino messages.
// Roles eino has no use for here (reasoning/activity) are skipped.
func toEinoMessages(in []aguitypes.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(in))
	for _, m := range in {
		content, ok := m.ContentString()
		switch m.Role {
		case aguitypes.RoleUser:
			// ContentString returns ok=false for multimodal/structured content
			// ([]InputContent). Skip rather than inject an empty user turn — a
			// blank message would silently drop the user's actual input.
			if !ok {
				continue
			}
			out = append(out, schema.UserMessage(content))
		case aguitypes.RoleSystem, aguitypes.RoleDeveloper:
			if !ok {
				continue
			}
			out = append(out, schema.SystemMessage(content))
		case aguitypes.RoleAssistant:
			out = append(out, &schema.Message{
				Role:      schema.Assistant,
				Content:   content,
				ToolCalls: toEinoToolCalls(m.ToolCalls),
			})
		case aguitypes.RoleTool:
			out = append(out, schema.ToolMessage(content, m.ToolCallID))
		}
	}
	return out
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
