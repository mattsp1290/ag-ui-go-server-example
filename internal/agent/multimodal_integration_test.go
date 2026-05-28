//go:build integration

package agent

// Run with: go test -tags integration -run TestMultimodalIntegration ./internal/agent/
// Requires OPENAI_API_KEY in the environment. Override the model with MODEL=gpt-4o.
// gpt-4o-mini is used by default (cheapest vision-capable model).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"

	"github.com/mattsp1290/ag-ui-go-server-example/internal/config"
	"github.com/mattsp1290/ag-ui-go-server-example/internal/runstore"
)

// TestMultimodalIntegration drives a full Run() against the real OpenAI API with
// an inline image, confirming that the multimodal forwarding path in convert.go
// is wired correctly end-to-end. A RUN_ERROR or missing TEXT_MESSAGE_CONTENT
// indicates the image was dropped before reaching the model.
func TestMultimodalIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	ctx := context.Background()
	base, err := NewModel(ctx, config.Config{Provider: "openai", Model: modelName})
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}

	tools, err := NewReadOnlyToolset(t.TempDir())
	if err != nil {
		t.Fatalf("NewReadOnlyToolset: %v", err)
	}
	boundModel, err := base.WithTools(tools.Infos())
	if err != nil {
		t.Fatalf("WithTools: %v", err)
	}

	in := &aguitypes.RunAgentInput{
		ThreadID: "t-multimodal",
		RunID:    "r-multimodal",
		Messages: []aguitypes.Message{
			{
				ID:   "msg-1",
				Role: aguitypes.RoleUser,
				// []InputContent exercises toEinoUserMessage → toEinoImagePart in convert.go.
				Content: []aguitypes.InputContent{
					{
						Type: aguitypes.InputContentTypeText,
						Text: "What color is this image? Reply with exactly one word.",
					},
					{
						Type: aguitypes.InputContentTypeImage,
						Source: &aguitypes.InputContentSource{
							Type:     aguitypes.InputContentSourceTypeData,
							Value:    redPNG1x1(t),
							MimeType: "image/png",
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	emit := NewEmitter(ctx, w, sse.NewSSEWriter(), in.ThreadID, in.RunID, nil)
	deps := &Deps{
		Model:         boundModel,
		Tools:         tools,
		Store:         runstore.New(),
		AutoApprove:   true,
		MaxIterations: 2,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Provider:      "openai",
	}

	Run(ctx, emit, in, deps, in.ThreadID, in.RunID)
	_ = w.Flush()
	out := buf.String()

	if strings.Contains(out, `"type":"RUN_ERROR"`) {
		t.Errorf("multimodal run produced RUN_ERROR — image may have been dropped or rejected:\n%s", out)
	}
	if !strings.Contains(out, `"type":"RUN_FINISHED"`) {
		t.Errorf("expected RUN_FINISHED:\n%s", out)
	}
	if !strings.Contains(out, `"type":"TEXT_MESSAGE_CONTENT"`) {
		t.Errorf("expected TEXT_MESSAGE_CONTENT — model should have responded to the image:\n%s", out)
	}
}

// redPNG1x1 generates a 1×1 red PNG and returns its raw base64 encoding.
// Inline generation avoids any external image-host dependency.
func redPNG1x1(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
