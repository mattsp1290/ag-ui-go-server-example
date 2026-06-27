package agent

import (
	"bufio"
	"context"

	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	sharedconvert "github.com/mattsp1290/eino-agui/convert"
	sharedemitter "github.com/mattsp1290/eino-agui/emitter"
	sharedstream "github.com/mattsp1290/eino-agui/stream"
	sharedtools "github.com/mattsp1290/eino-agui/tools"
)

type Emitter = sharedemitter.Emitter

func NewEmitter(ctx context.Context, w *bufio.Writer, sw *sse.SSEWriter, threadID, runID string, cancel context.CancelFunc) *Emitter {
	return sharedemitter.NewEmitter(ctx, w, sw, threadID, runID, cancel)
}

func toEinoMessages(in []aguitypes.Message, provider string) []*schema.Message {
	return sharedconvert.ToEinoMessages(in, sharedconvert.WithVisionSupport(supportsVision(provider)))
}

func supportsVision(provider string) bool {
	return provider == "openai"
}

func messageText(m aguitypes.Message) string {
	return sharedconvert.MessageText(m)
}

func toAGUIMessages(msgs []*schema.Message) []aguitypes.Message {
	return sharedconvert.ToAGUIMessages(msgs)
}

func streamTurn(ctx context.Context, emit *Emitter, cm model.ToolCallingChatModel, messages []*schema.Message, liveToolCalls bool) (*schema.Message, error) {
	return sharedstream.StreamTurn(ctx, emit, cm, messages, sharedstream.WithLiveToolCallEvents(liveToolCalls))
}

func clientToolInfos(tools []aguitypes.Tool) ([]*schema.ToolInfo, error) {
	return sharedtools.ClientToolInfos(tools, sharedtools.WithUnsupportedSchemaKeywords())
}

func toJSONSchema(params any) (*jsonschema.Schema, error) {
	return sharedtools.ToJSONSchema(params)
}

func classifyToolCalls(calls []schema.ToolCall, clientNames map[string]bool) (server, client []schema.ToolCall) {
	return sharedtools.ClassifyToolCalls(calls, clientNames)
}
