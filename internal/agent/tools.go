package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mattsp1290/eino-tools/fileops"
)

// Toolset is the agent's tool registry.
//
// NO-FILE-WRITE POLICY: this agent must never modify the filesystem. Only the
// read-only fileops.ReadTool ("file_read") is registered. Do NOT add
// fileops.NewWriteTool / NewEditTool, the shell tool, or trackerwrite here — the
// whole point of this server is a read-only agent. The eino-tools workspace root
// further sandboxes reads to a single directory.
type Toolset struct {
	infos  []*schema.ToolInfo
	byName map[string]tool.InvokableTool
}

// NewReadOnlyToolset builds a registry containing only file_read, rooted at the
// given absolute workspace directory.
func NewReadOnlyToolset(workspace string) (*Toolset, error) {
	rt, err := fileops.NewReadTool(workspace)
	if err != nil {
		return nil, fmt.Errorf("file_read tool: %w", err)
	}
	info, err := rt.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("file_read info: %w", err)
	}
	return &Toolset{
		infos:  []*schema.ToolInfo{info},
		byName: map[string]tool.InvokableTool{info.Name: rt},
	}, nil
}

// Infos returns the tool schemas to bind to the chat model.
func (t *Toolset) Infos() []*schema.ToolInfo { return t.infos }

// Run executes a registered tool by name with JSON arguments.
func (t *Toolset) Run(ctx context.Context, name, argsJSON string) (string, error) {
	tl, ok := t.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown or unpermitted tool %q (this agent only allows file_read)", name)
	}
	return tl.InvokableRun(ctx, argsJSON)
}
