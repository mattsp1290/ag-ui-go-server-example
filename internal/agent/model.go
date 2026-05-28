package agent

import (
	"context"
	"fmt"
	"os"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/mattsp1290/eino-providers/openaicodex"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/config"
)

// NewModel constructs the eino chat model for the configured provider.
//
//   - "openai-codex" (default): the ChatGPT subscription backend from eino-providers,
//     authenticated via codex-auth-go. This is the target path.
//   - "openai": a plain OPENAI_API_KEY harness (Chat Completions) for offline/local
//     testing without a subscription login.
//
// Both satisfy model.ToolCallingChatModel, so the agent loop is backend-agnostic.
func NewModel(ctx context.Context, cfg config.Config) (model.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case "openai-codex", "":
		return openaicodex.NewChatModel(ctx, openaicodex.ChatModelConfig{
			AppName: cfg.CodexAppName,
			Model:   cfg.Model,
		})
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("openai provider requires OPENAI_API_KEY")
		}
		return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
			APIKey: key,
			Model:  cfg.Model,
		})
	default:
		return nil, fmt.Errorf("unknown MODEL_PROVIDER %q (want openai-codex or openai)", cfg.Provider)
	}
}
