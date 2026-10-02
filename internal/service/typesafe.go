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
)

type TypeSafe struct {
	logger  *Logger
	client  *http.Client
	baseURL string
	apiKey  string
}

type TypeSafeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type typeSafeRequest struct {
	State     []TypeSafeMessage                 `json:"state"`
	Model     string                            `json:"model"`
	Questions map[string]typeSafeChoiceQuestion `json:"questions"`
}

type typeSafeChoiceQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type typeSafeResponse struct {
	Answers map[string]struct {
		Type   string `json:"type"`
		Choice string `json:"choice"`
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
func (typeSafe *TypeSafe) DetectIntent(ctx context.Context, messages []TypeSafeMessage, intents []string) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages must not be empty")
	}
	if len(intents) == 0 {
		return "", fmt.Errorf("intents must not be empty")
	}
	if typeSafe == nil || typeSafe.client == nil {
		return "", fmt.Errorf("Jev client is not initialized")
	}
	if strings.TrimSpace(typeSafe.apiKey) == "" {
		return "", fmt.Errorf("Jev API key is not configured")
	}
	criteria := make(map[string]string, len(intents))
	for _, intent := range intents {
		intent = strings.TrimSpace(intent)
		if intent != "" {
			criteria[intent] = "The conversation matches this intent."
		}
	}
	if len(criteria) == 0 {
		return "", fmt.Errorf("intents must contain at least one non-empty value")
	}
	requestBody := typeSafeRequest{
		State: messages,
		Model: "jev-latest",
		Questions: map[string]typeSafeChoiceQuestion{
			"detected_intent": {
				Type:         "choice",
				Instructions: "Which intent best matches this conversation?",
				Criteria:     criteria,
			},
		},
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal Jev request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, typeSafe.baseURL+"/v1/systemone", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create Jev request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+typeSafe.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := typeSafe.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("call Jev: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read Jev response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Jev returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result typeSafeResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode Jev response: %w", err)
	}
	answer, ok := result.Answers["detected_intent"]
	if !ok || answer.Type != "choice" || strings.TrimSpace(answer.Choice) == "" {
		return "", fmt.Errorf("Jev response did not contain a detected intent")
	}
	return answer.Choice, nil
}
