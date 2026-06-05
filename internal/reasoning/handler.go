package reasoning

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"github.com/gofiber/fiber/v3"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/agent"
)

// Handler returns a Fiber handler for POST /reasoning.
// It emits a scripted REASONING_* event sequence followed by a TEXT response,
// exercising the full reasoning protocol surface without requiring a live LLM.
func Handler(shutdownCtx context.Context, logger *slog.Logger) fiber.Handler {
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

		return c.SendStreamWriter(func(w *bufio.Writer) {
			runCtx, cancel := context.WithCancel(shutdownCtx)
			defer cancel()
			emit := agent.NewEmitter(runCtx, w, sw, threadID, runID, cancel)

			defer func() {
				if r := recover(); r != nil {
					logger.Error("reasoning handler panicked", "thread", threadID, "run", runID, "panic", r)
					emit.RunError("the reasoning handler crashed")
				}
			}()

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
		})
	}
}
