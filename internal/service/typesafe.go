package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/types"
)

type TypeSafe struct {
	logger  *Logger
	client  *http.Client
	baseURL string
	apiKey  string
}

type typeSafeRequest struct {
	State     []typeSafeMessage                 `json:"state"`
	Model     string                            `json:"model"`
	Questions map[string]TypeSafeChoiceQuestion `json:"questions"`
}

type typeSafeMessage struct {
	Role    types.AIMessageRole `json:"role"`
	Content string              `json:"content"`
}

type TypeSafeChoiceQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type TypeSafeResponse struct {
	Answers map[string]struct {
		Type   string  `json:"type"`
		Choice string  `json:"choice"`
		Noul   float32 `json:"noul"`
	} `json:"answers"`
}

func NewTypeSafe(logger *Logger) *TypeSafe {
	return &TypeSafe{
		logger:  logger,
		client:  &http.Client{},
		baseURL: "https://api.typesafe.ai",
		apiKey:  cfg.Default().Typesafe.JevAPIKey,
	}
}

// DetectIntent asks Jev to select one intent for a multi-turn conversation.
func (typeSafe *TypeSafe) DetectChoice(ctx context.Context, messages []dto_ai.Message, questions map[string]TypeSafeChoiceQuestion) (*TypeSafeResponse, error) {
	var chatMessages []typeSafeMessage
	for _, m := range messages {
		chatMessages = append(chatMessages, typeSafeMessage{
			Role:    m.Role,
			Content: m.Text(),
		})
	}
	requestBody := typeSafeRequest{
		State:     chatMessages,
		Model:     "jev-latest",
		Questions: questions,
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal Jev request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, typeSafe.baseURL+"/v1/systemone", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Jev request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+typeSafe.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := typeSafe.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call Jev: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read Jev response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Jev returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result TypeSafeResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode Jev response: %w", err)
	}
	return &result, nil
}
