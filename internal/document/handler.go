package document

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

const stubMessage = "Document Q&A is coming in a future release."

// Handler returns a Fiber handler for POST /document.
// This is a stub that validates the Flutter multimodal document input path.
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
					logger.Error("document handler panicked", "thread", threadID, "run", runID, "panic", r)
					emit.RunError("the document handler crashed")
				}
			}()

			emit.RunStarted()

			if !hasDocumentPart(in.Messages) {
				emit.RunError("document: no document part found in the last user message")
				return
			}

			msgID := aguievents.GenerateMessageID()
			emit.TextStart(msgID)
			emit.TextContent(msgID, stubMessage)
			emit.TextEnd(msgID)
			emit.MessagesSnapshot([]aguitypes.Message{})
			emit.RunFinishedSuccess()
		})
	}
}

// hasDocumentPart reports whether the last user message contains a document InputContent
// with a DataSource. Validates the Flutter path without requiring a real PDF parser.
func hasDocumentPart(messages []aguitypes.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != aguitypes.RoleUser {
			continue
		}
		parts, hasParts := m.ContentInputContents()
		if !hasParts {
			continue
		}
		for _, p := range parts {
			if p.Type == aguitypes.InputContentTypeDocument &&
				p.Source != nil &&
				p.Source.Type == aguitypes.InputContentSourceTypeData &&
				p.Source.Value != "" {
				return true
			}
		}
	}
	return false
}
