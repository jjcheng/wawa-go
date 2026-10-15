package dto

import "time"

type SearchResponse interface {
	Base() SearchResponseBase
	ReturnFields(withEmbedding bool) []string
	Payload() (map[string]any, error)
	NewObject(payload map[string]any) (SearchResponse, error)
}

type SearchResponseBase struct {
	Id            string    `json:"id"`
	AddedAt       time.Time `json:"added_at"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}
