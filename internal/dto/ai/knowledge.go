package dto_ai

import (
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Knowledge struct {
	dto.SearchResponseBase
	Type           types.AIKnowledgeType     `json:"type" val:"required" description:"TEXT, IMAGE, DOCUMENT, VIDEO" example:"TEXT"`
	Category       types.AIKnowledgeCategory `json:"category" val:"required" description:"FAQ"`
	ProfileId      int32                     `json:"profile_id" val:"required" description:"id of the agent profile"`
	Title          string                    `json:"title" val:"required" description:"title of the knowledge"`
	Body           string                    `json:"body" val:"required" description:"body of the knowledge"`
	TitleEmbedding []float32                 `json:"title_embedding,omitempty" val:"required" description:"embedding of the title"`
	BodyEmbedding  []float32                 `json:"body_embedding,omitempty" val:"required" description:"embedding of the body"`
	Enabled        bool                      `json:"enabled" description:"if it should be searched or not"`
	//not stored
	Score  float32 `json:"score,omitempty"`
	Tokens int     `json:"tokens,omitempty"` // used when creating calcuate total tokens
}

func (knowledge Knowledge) Base() dto.SearchResponseBase {
	return knowledge.SearchResponseBase
}

func (Knowledge) ReturnFields(withEmbedding bool) []string {
	list := []string{"id", "profile_id", "type", "category", "title", "body", "enabled", "added_at", "last_updated_at"}
	if withEmbedding {
		list = append(list, "title_embedding", "body_embedding")
	}
	return list
}

func (knowledge Knowledge) Payload() (map[string]any, error) {
	dic := map[string]any{
		"id":              knowledge.Base().Id,
		"profile_id":      knowledge.ProfileId,
		"type":            knowledge.Type,
		"category":        knowledge.Category,
		"title":           knowledge.Title,
		"body":            knowledge.Body,
		"enabled":         knowledge.Enabled,
		"added_at":        knowledge.Base().AddedAt,
		"last_updated_at": knowledge.Base().LastUpdatedAt,
	}
	if len(knowledge.TitleEmbedding) > 0 {
		dic["title_embedding"] = knowledge.TitleEmbedding
	}
	if len(knowledge.BodyEmbedding) > 0 {
		dic["body_embedding"] = knowledge.BodyEmbedding
	}
	return dic, nil
}

func (Knowledge) NewObject(payload map[string]any) (dto.SearchResponse, error) {
	added_at, err := helper.ConvertStringToTime(payload["added_at"].(string))
	if err != nil {
		return nil, err
	}
	lastUpdatedAt, err := helper.ConvertStringToTime(payload["last_updated_at"].(string))
	if err != nil {
		return nil, err
	}
	knowledge := Knowledge{
		SearchResponseBase: dto.SearchResponseBase{
			Id:            payload["id"].(string),
			AddedAt:       *added_at,
			LastUpdatedAt: *lastUpdatedAt,
		},
		ProfileId: int32(payload["profile_id"].(float64)),
		Type:      types.AIKnowledgeType(payload["type"].(string)),
		Category:  types.AIKnowledgeCategory(payload["category"].(string)),
		Title:     payload["title"].(string),
		Body:      payload["body"].(string),
		Enabled:   payload["enabled"].(bool),
	}
	if score, ok := payload["score"].(float32); ok {
		knowledge.Score = score
	}
	if bodyEmbedding, ok := payload["body_embedding"].([]any); ok && len(bodyEmbedding) > 0 {
		knowledge.BodyEmbedding = helper.Map(bodyEmbedding, func(a any) float32 {
			return float32(a.(float64))
		})
	}
	if titleEmbedding, ok := payload["title_embedding"].([]any); ok && len(titleEmbedding) > 0 {
		knowledge.TitleEmbedding = helper.Map(titleEmbedding, func(a any) float32 {
			return float32(a.(float64))
		})
	}
	return knowledge, nil
}
