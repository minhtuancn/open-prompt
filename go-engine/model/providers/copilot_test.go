package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCopilotImplementsProvider(t *testing.T) {
	var _ Provider = (*CopilotProvider)(nil)
}

func TestCopilotStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth=%q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Editor-Version") == "" {
			t.Error("missing Editor-Version header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"copilot says hi\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	cp := NewCopilotProviderWithBaseURL("test-token", srv.URL)
	var chunks []string
	err := cp.StreamComplete(context.Background(), CompletionRequest{Prompt: "hello"}, func(s string) {
		chunks = append(chunks, s)
	})

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "copilot says hi" {
		t.Errorf("chunks=%v", chunks)
	}
}

func TestCopilotError403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"error":"no copilot subscription"}`)
	}))
	defer srv.Close()

	cp := NewCopilotProviderWithBaseURL("bad-token", srv.URL)
	err := cp.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
