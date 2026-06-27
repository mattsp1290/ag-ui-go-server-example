package agent

import (
	"strings"
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
	}
}

const agenticChatSystemPrompt = "You are a helpful assistant. When a user request is best " +
	"served by one of the available tools, call that tool with arguments that conform to its " +
	"schema. You do not execute those tools yourself — you propose the call and the client " +
	"fulfills it, then continues the conversation with the result. If no tool fits, answer " +
	"directly and concisely."

// ToolBasedGenerativeUIConfig is the /tool_based_generative_ui posture: prefer the
// generative-UI rendering tool over a prose answer for fitting prompts. It is 01's
// machinery (client tools + streaming tap) with a stronger prefer-the-tool prompt.
func ToolBasedGenerativeUIConfig() RunConfig {
	cfg := AgenticChatConfig()
	cfg.SystemPrompt = genUISystemPrompt
	return cfg
}

const genUISystemPrompt = "You are a generative-UI assistant. When the user's request can be " +
	"presented through one of the provided rendering tools (for example a tool that renders " +
	"structured content such as a card), you MUST call that tool with well-formed structured " +
	"arguments that satisfy its schema, rather than answering in prose. Only answer in plain " +
	"text when no provided tool fits the request."

// HumanInTheLoopConfig is the /human_in_the_loop posture: route consequential
// actions through a client-defined approval tool and wait for the user's decision
// (carried back as a role:tool result on the follow-up run). It is 01's machinery
// with an approval-gating prompt. The per-request approval value ("off" disables
// the gate, serving plain agentic chat; anything else — including empty — keeps it
// on) comes from the X-AG-Approval header or the ?approval= query param.
func HumanInTheLoopConfig(approval string) RunConfig {
	cfg := AgenticChatConfig()
	if strings.EqualFold(approval, "off") {
		return cfg // ungated: behave like /agentic_chat
	}
	cfg.SystemPrompt = humanInTheLoopSystemPrompt
	return cfg
}

const humanInTheLoopSystemPrompt = "You are a careful assistant. Before performing any " +
	"consequential or irreversible action (deleting, sending, purchasing, or modifying data), " +
	"you MUST first call the provided approval tool with a clear, human-readable summary of " +
	"what you intend to do, and wait for the result. Proceed only after the user approves; if " +
	"the user rejects, acknowledge it and do not perform the action. For non-consequential " +
	"requests, answer directly."
