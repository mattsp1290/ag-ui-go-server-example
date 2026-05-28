// Command server is a Fiber v3 implementation of the AG-UI protocol backed by an
// eino agent loop. The agent streams the full AG-UI event surface and can read
// (never write) files via a single read-only file_read tool.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/agent"
	"github.com/mattsp1290/ag-ui-go-server-example/internal/config"
	"github.com/mattsp1290/ag-ui-go-server-example/internal/runstore"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	ctx := context.Background()

	base, err := agent.NewModel(ctx, cfg)
	if err != nil {
		logger.Error("failed to construct chat model", "provider", cfg.Provider, "error", err)
		fmt.Fprintf(os.Stderr, "\nHint: for openai-codex, log in with codex-auth-go for app name %q first.\n", cfg.CodexAppName)
		os.Exit(1)
	}

	tools, err := agent.NewReadOnlyToolset(cfg.Workspace)
	if err != nil {
		logger.Error("failed to build toolset", "workspace", cfg.Workspace, "error", err)
		os.Exit(1)
	}

	// boundModel is shared across all concurrent /agentic requests. This is safe
	// because eino's ToolCallingChatModel.Stream is stateless per call (it takes
	// the messages as an argument and returns a fresh StreamReader); the codex
	// provider holds no mutable per-request state. A future provider that caches
	// per-instance state would need a per-request clone or pool.
	boundModel, err := base.WithTools(tools.Infos())
	if err != nil {
		logger.Error("failed to bind tools", "error", err)
		os.Exit(1)
	}

	deps := &agent.Deps{
		Model:         boundModel,
		Tools:         tools,
		Store:         runstore.New(),
		AutoApprove:   cfg.AutoApprove,
		MaxIterations: cfg.MaxIterations,
		Logger:        logger,
	}

	app := fiber.New(fiber.Config{AppName: "ag-ui-go-server-example", BodyLimit: 4 * 1024 * 1024})
	app.Use(requestid.New())
	if cfg.CORS {
		app.Use(cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Cache-Control"},
		}))
	}

	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":     "ag-ui-go-server-example is running",
			"provider":    cfg.Provider,
			"model":       cfg.Model,
			"workspace":   cfg.Workspace,
			"autoApprove": cfg.AutoApprove,
		})
	})

	app.Post("/agentic", agenticHandler(deps, logger))

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	logger.Info("starting server", "addr", addr, "provider", cfg.Provider, "model", cfg.Model,
		"workspace", cfg.Workspace, "autoApprove", cfg.AutoApprove)
	if err := app.Listen(addr); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func agenticHandler(deps *agent.Deps, logger *slog.Logger) fiber.Handler {
	sw := sse.NewSSEWriter().WithLogger(logger)
	return func(c fiber.Ctx) error {
		var in aguitypes.RunAgentInput
		if err := json.Unmarshal(c.Body(), &in); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		threadID := in.ThreadID
		if threadID == "" {
			threadID = aguievents.GenerateThreadID()
		}
		runID := in.RunID
		if runID == "" {
			runID = aguievents.GenerateRunID()
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Access-Control-Allow-Origin", "*")

		return c.SendStreamWriter(func(w *bufio.Writer) {
			// Derive the run context from Background, not the fasthttp RequestCtx:
			// the stream writer runs after the handler returns (RequestCtx is then
			// recycled) and RequestCtx never signals client disconnect. The emitter
			// cancels this context on the first failed write, aborting the model
			// stream when the client goes away.
			runCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			emit := agent.NewEmitter(runCtx, w, sw, threadID, runID, cancel)
			agent.Run(runCtx, emit, &in, deps, threadID, runID)
			if err := emit.Err(); err != nil {
				logger.Warn("event stream ended early", "thread", threadID, "run", runID, "error", err)
			}
		})
	}
}
