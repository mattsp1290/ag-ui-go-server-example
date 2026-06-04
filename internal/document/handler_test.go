package document

import (
	"testing"

	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

func makeUserMsg(parts []aguitypes.InputContent) aguitypes.Message {
	return aguitypes.Message{Role: aguitypes.RoleUser, Content: parts}
}

func TestHasDocumentPart(t *testing.T) {
	docPart := func(b64 string) aguitypes.InputContent {
		return aguitypes.InputContent{
			Type: aguitypes.InputContentTypeDocument,
			Source: &aguitypes.InputContentSource{
				Type:     aguitypes.InputContentSourceTypeData,
				Value:    b64,
				MimeType: "application/pdf",
			},
		}
	}
	textPart := func(t string) aguitypes.InputContent {
		return aguitypes.InputContent{Type: aguitypes.InputContentTypeText, Text: t}
	}

	tests := []struct {
		name     string
		messages []aguitypes.Message
		want     bool
	}{
		{
			name: "no messages",
			want: false,
		},
		{
			name: "text-only message",
			messages: []aguitypes.Message{
				makeUserMsg([]aguitypes.InputContent{textPart("hello")}),
			},
			want: false,
		},
		{
			name: "document part found",
			messages: []aguitypes.Message{
				makeUserMsg([]aguitypes.InputContent{
					docPart("JVBERi0xLjQK"),
					textPart("What is this?"),
				}),
			},
			want: true,
		},
		{
			name: "URL-source document is skipped",
			messages: []aguitypes.Message{
				makeUserMsg([]aguitypes.InputContent{
					{
						Type: aguitypes.InputContentTypeDocument,
						Source: &aguitypes.InputContentSource{
							Type:  aguitypes.InputContentSourceTypeURL,
							Value: "https://example.com/doc.pdf",
						},
					},
				}),
			},
			want: false,
		},
		{
			name: "non-user message is skipped",
			messages: []aguitypes.Message{
				{
					Role:    aguitypes.RoleAssistant,
					Content: []aguitypes.InputContent{docPart("abc")},
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasDocumentPart(tc.messages)
			if got != tc.want {
				t.Errorf("hasDocumentPart()=%v want %v", got, tc.want)
			}
		})
	}
}
