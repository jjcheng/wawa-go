package repository

import (
	"context"

	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/types"
)

type AIKnowledgeRepository interface {
	SearchRepository[dto_ai.Knowledge]
	Search(ctx context.Context, query string, embedding []float32, profileIds []int32, topK int, threshold float32, useSemantic bool, vectorFiedsToSearch []string, vectorFieldWeights []float32) ([]dto_ai.Knowledge, error)
	ListByProfileId(ctx context.Context, profileId int32, category types.AIKnowledgeCategory) ([]dto_ai.Knowledge, error)
	Get(ctx context.Context, id string, returnEmbedding bool) (*dto_ai.Knowledge, error)
	GetByBody(ctx context.Context, body string) (*dto_ai.Knowledge, error)
}
