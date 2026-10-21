package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTargetFor(t *testing.T) {
	for _, tc := range []struct {
		text string
		want language
	}{
		{"你好，世界", english},
		{"這是繁體中文", english},
		{"我在用 React 写代码", english},
		{"这个 API 的 latency 太高了", english},
		{"Hello, world", chinese},
		{"I love 北京", chinese},
		{"日本語のテキストです", chinese},
		{"안녕하세요", chinese},
		{"Bonjour tout le monde", chinese},
		{"12345", chinese},
	} {
		if got := targetFor(tc.text); got != tc.want {
			t.Errorf("targetFor(%q) = %s, want %s", tc.text, got.name, tc.want.name)
		}
	}
}

// fakeDeepSeek serves a streamed reply of pieces, and records the request.
func fakeDeepSeek(t *testing.T, got *chatRequest, pieces ...string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-test" {
			http.Error(w, `{"error":{"message":"bad request"}}`, http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, got); err != nil {
			t.Errorf("request body %s: %v", body, err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, ": keep-alive\n\n")
		for _, p := range pieces {
			chunk, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": p}}}})
			io.WriteString(w, "data: "+string(chunk)+"\n\n")
		}
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTranslateStreams(t *testing.T) {
	var req chatRequest
	srv := fakeDeepSeek(t, &req, "Hello", ", ", "world")
	var out strings.Builder
	err := deepseek{baseURL: srv.URL}.translate(context.Background(), "sk-test", "你好，世界", english, func(s string) { out.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello, world" {
		t.Errorf("translation %q", out.String())
	}
	if req.Model != model || !req.Stream || req.Thinking.Type != "disabled" || len(req.Messages) != 2 ||
		!strings.Contains(req.Messages[0].Content, "into English") || req.Messages[1].Content != "你好，世界" {
		t.Errorf("request %+v", req)
	}
}

func TestTranslateRejected(t *testing.T) {
	srv := fakeDeepSeek(t, new(chatRequest))
	err := deepseek{baseURL: srv.URL}.translate(context.Background(), "sk-wrong", "hi", chinese, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "API Key 无效") {
		t.Errorf("error %v, want an invalid key", err)
	}
}
