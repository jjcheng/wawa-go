package dto_ai_worker

import (
	"encoding/json"
	"strings"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	Feature        string                    `json:"feature"`
	Role           types.AIWorkerMessageRole `json:"role"`
	Parts          []WorkResultPart          `json:"parts"`
	ConversationId int32                     `json:"conversation_id"`
	URL            string                    `json:"url"`
}

func NewMessage(message dao_ai_worker.Message) Message {
	parts := make([]WorkResultPart, 0, len(message.Parts))
	for _, serializedPart := range message.Parts {
		var part WorkResultPart
		err := json.Unmarshal([]byte(serializedPart), &part)
		// Parts no longer require a type, so only fall back to text for non-part strings.
		if err != nil || (part.Input == nil && part.Content == nil) {
			continue
		}
		parts = append(parts, part)
	}
	return Message{
		DTOBase: dto.DTOBase{
			Id:            message.Id,
			AddedAt:       message.AddedAt,
			LastUpdatedAt: message.LastUpdatedAt,
		},
		Feature:        message.Feature,
		ConversationId: message.ConversationId,
		Role:           message.Role,
		Parts:          parts,
		URL:            message.URL,
	}
}

func SerializeWorkResultParts(parts []WorkResultPart) ([]string, error) {
	content := make([]string, 0, len(parts))
	for _, part := range parts {
		serializedPart, err := json.Marshal(part)
		if err != nil {
			return nil, err
		}
		content = append(content, string(serializedPart))
	}
	return content, nil
}

func (message Message) Text() string {
	var texts []string
	// for _, part := range message.Parts {
	// 	switch part.Type {
	// 	case types.AIWorkResultPartTypeText:
	// 		if text, ok := part.Content.(string); ok {
	// 			texts = append(texts, text)
	// 		}
	// 	default:
	// 		texts = append(texts, "data is successfully retrieved but ommitted here")
	// 	}
	// }
	for _, part := range message.Parts {
		if text, ok := part.Content.(string); ok {
			if (strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]")) ||
				(strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}")) {
				texts = append(texts, "data is successfully retrieved but ommitted here")
			} else {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, "\n\n")
}
