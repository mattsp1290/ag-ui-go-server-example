// Package config loads server configuration from the environment.
package config

import (
	"os"
	"strconv"
)

// Config holds the server's runtime configuration.
type Config struct {
	Host string
	Port int

	// Provider selects the eino chat-model backend: "openai-codex" (ChatGPT
	// subscription, default) or "openai" (plain OPENAI_API_KEY harness).
	Provider string
	// Model is the model slug passed to the provider.
	Model string
	// CodexAppName is the codex-auth-go credential app name for the subscription.
	CodexAppName string

	// Workspace is the absolute directory the read-only file_read tool is rooted at.
	// Defaults to the process working directory (override with AGENT_WORKSPACE);
	// since this is the root the agent can read, set it deliberately in real use.
	Workspace string
	// AutoApprove bypasses the human-in-the-loop tool-approval interrupt when true.
	AutoApprove bool
	// MaxIterations bounds the model<->tool loop per run (env AGENT_MAX_ITERATIONS).
	MaxIterations int
	// CORS enables permissive CORS for local UI development.
	CORS bool
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	wd, _ := os.Getwd()
	provider := envOr("MODEL_PROVIDER", "openai-codex")
	model := os.Getenv("MODEL")
	if model == "" {
		if provider == "openai" {
			model = "gpt-4o"
		} else {
			model = "gpt-5.5" // codex CLI default
		}
	}
	return Config{
		Host:          envOr("HOST", "127.0.0.1"),
		Port:          envInt("PORT", 8080),
		Provider:      provider,
		Model:         model,
		CodexAppName:  envOr("CODEX_APP_NAME", "ag-ui-go-server-example"),
		Workspace:     envOr("AGENT_WORKSPACE", wd),
		AutoApprove:   envBool("AGENT_AUTO_APPROVE", false),
		MaxIterations: envInt("AGENT_MAX_ITERATIONS", 8),
		CORS:          envBool("CORS_ENABLED", true),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
