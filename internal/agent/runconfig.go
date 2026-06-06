package agent

import (
	"encoding/json"
	"fmt"

	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// ToolPolicy selects how a route treats tools.
type ToolPolicy int

const (
	// ServerOnly is today's /agentic behavior: only server-owned tools (file_read).
	ServerOnly ToolPolicy = iota
	// ClientTools accepts RunAgentInput.tools and hands client-defined calls back
	// to the client (the server cannot execute them).
	ClientTools
)

// StateModel selects which state surface a route uses.
type StateModel int

const (
	// LifecycleState is the agent-lifecycle state (status/filesRead/toolCalls).
	LifecycleState StateModel = iota
	// NoAgentState means the route owns its own state emission (no lifecycle state).
	NoAgentState
)

// RunConfig parameterizes a single route's behavior over the shared agent loop.
// The zero value is the /agentic default (ServerOnly, lifecycle state, interrupt
// per AutoApprove, post-turn emitToolProposal).
type RunConfig struct {
	// SystemPrompt is the route's posture. Empty uses the default file_read prompt.
	SystemPrompt string
	// ToolPolicy selects server-only vs client-defined tool handling.
	ToolPolicy ToolPolicy
	// ExposeFileRead binds the server file_read tool on this route.
	ExposeFileRead bool
	// NeverInterrupt skips the approval-interrupt branch entirely (feature routes
	// can't resume — the Dart client has no resume path).
	NeverInterrupt bool
	// StreamToolCalls streams TOOL_CALL_* live from streamTurn instead of via the
	// post-turn emitToolProposal. The two are mutually exclusive (never both, or
	// every call double-emits). Only the tools track sets this.
	StreamToolCalls bool
	// StateModel selects the state surface.
	StateModel StateModel
}

// DefaultRunConfig is /agentic's behavior: unchanged from before the refactor.
func DefaultRunConfig() RunConfig { return RunConfig{} }

// AgenticChatConfig is the /agentic_chat (frontend-tools) posture.
func AgenticChatConfig() RunConfig {
	return RunConfig{
		SystemPrompt:    agenticChatSystemPrompt,
		ToolPolicy:      ClientTools,
		ExposeFileRead:  false,
		NeverInterrupt:  true,
		StreamToolCalls: true,
		StateModel:      LifecycleState,
	}
}

const agenticChatSystemPrompt = "You are a helpful assistant. When a user request is best " +
	"served by one of the available tools, call that tool with arguments that conform to its " +
	"schema. You do not execute those tools yourself — you propose the call and the client " +
	"fulfills it, then continues the conversation with the result. If no tool fits, answer " +
	"directly and concisely."

// clientToolInfos converts AG-UI client tool definitions (RunAgentInput.tools)
// into eino ToolInfos the model can be bound to. Errors (empty/duplicate names,
// unparseable parameter schemas) are surfaced as RUN_ERROR by the caller.
func clientToolInfos(tools []aguitypes.Tool) ([]*schema.ToolInfo, error) {
	out := make([]*schema.ToolInfo, 0, len(tools))
	seen := make(map[string]bool, len(tools))
	for _, t := range tools {
		if t.Name == "" {
			return nil, fmt.Errorf("a client tool has an empty name")
		}
		if seen[t.Name] {
			return nil, fmt.Errorf("duplicate client tool name %q", t.Name)
		}
		seen[t.Name] = true

		info := &schema.ToolInfo{Name: t.Name, Desc: t.Description}
		js, err := toJSONSchema(t.Parameters)
		if err != nil {
			return nil, fmt.Errorf("tool %q parameters: %w", t.Name, err)
		}
		if js != nil {
			info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(js)
		}
		out = append(out, info)
	}
	return out, nil
}

// toJSONSchema converts an arbitrary client-supplied JSON Schema (the tool's
// `parameters`, decoded as `any`) into an eino *jsonschema.Schema. A nil/absent
// schema yields nil (a no-argument tool).
func toJSONSchema(params any) (*jsonschema.Schema, error) {
	if params == nil {
		return nil, nil
	}
	b, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return nil, nil
	}
	var s jsonschema.Schema
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("not a valid JSON Schema: %w", err)
	}
	return &s, nil
}

// classifyToolCalls splits actionable calls into server-owned (executable here)
// and client-defined (handed back). An unknown name is routed to the server path,
// where settlePendingToolCalls answers it with an "unknown tool" error the model
// can recover from — never a panic.
func classifyToolCalls(calls []schema.ToolCall, serverTools *Toolset, clientNames map[string]bool) (server, client []schema.ToolCall) {
	for _, tc := range calls {
		switch {
		case clientNames[tc.Function.Name]:
			client = append(client, tc)
		default:
			server = append(server, tc)
		}
	}
	return server, client
}
