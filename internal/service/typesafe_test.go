package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTypeSafeDetectIntent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("request method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/v1/systemone" {
			t.Fatalf("request path = %s, want /v1/systemone", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q, want test bearer token", request.Header.Get("Authorization"))
		}

		var payload typeSafeRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "jev-latest" {
			t.Fatalf("model = %q, want jev-latest", payload.Model)
		}
		if len(payload.State) != 2 || payload.State[1].Content != "I need a refund" {
			t.Fatalf("state = %#v, want two multi-turn messages", payload.State)
		}
		question, ok := payload.Questions["detected_intent"]
		if !ok || question.Type != "choice" {
			t.Fatalf("detected intent question = %#v, want choice question", question)
		}
		if len(question.Criteria) != 2 || question.Criteria["refund"] == "" {
			t.Fatalf("criteria = %#v, want supplied intents", question.Criteria)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"answers":{"detected_intent":{"type":"choice","choice":"refund"}}}`))
	}))
	defer server.Close()

	client := &TypeSafe{
		client:  server.Client(),
		baseURL: server.URL,
		apiKey:  "test-key",
	}
	intent, err := client.DetectIntent(context.Background(), []TypeSafeMessage{
		{Role: "user", Content: "The order arrived damaged"},
		{Role: "user", Content: "I need a refund"},
	}, []string{"support", "refund"})
	if err != nil {
		t.Fatalf("DetectIntent() error = %v", err)
	}
	if intent != "refund" {
		t.Fatalf("DetectIntent() = %q, want refund", intent)
	}
}

func TestTypeSafeDetectIntentRejectsEmptyInput(t *testing.T) {
	client := &TypeSafe{apiKey: "test-key", baseURL: "https://api.typesafe.ai"}

	if _, err := client.DetectIntent(context.Background(), nil, []string{"support"}); err == nil {
		t.Fatal("DetectIntent() accepted empty messages")
	}
	if _, err := client.DetectIntent(context.Background(), []TypeSafeMessage{{Content: "hello"}}, nil); err == nil {
		t.Fatal("DetectIntent() accepted empty intents")
	}
}
