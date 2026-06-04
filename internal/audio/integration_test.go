//go:build integration

package audio

// Run with: go test -tags integration -run TestTranscribeIntegration ./internal/audio/
// Requires OPENAI_API_KEY in the environment.
//
// Test fixture: testdata/savior.wav
// Source: https://commons.wikimedia.org/wiki/File:LL-Q1860_(eng)-Wodencafe-savior.wav
// Author: Wodencafe (Lingua Libre contributor)
// License: CC0 1.0 Universal — https://creativecommons.org/publicdomain/zero/1.0/

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestTranscribeIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	raw, err := os.ReadFile("testdata/savior.wav")
	if err != nil {
		t.Fatalf("reading testdata/savior.wav: %v", err)
	}

	result, err := Transcribe(context.Background(), TranscribeRequest{
		AudioBase64: base64.StdEncoding.EncodeToString(raw),
		MimeType:    "audio/wav",
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if result.Text == "" {
		t.Fatal("expected non-empty transcription")
	}
	t.Logf("transcription: %q", result.Text)

	// Whisper should recognise the word "savior" (case-insensitive, may include punctuation)
	if !strings.Contains(strings.ToLower(result.Text), "savior") {
		t.Errorf("expected transcription to contain %q, got %q", "savior", result.Text)
	}
}
