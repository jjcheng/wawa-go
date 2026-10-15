package repository

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
)

type SearchRepository[T dto.SearchResponse] interface {
	// INFO
	GetIndexes(ctx context.Context) ([]string, error)
	GetIndex(ctx context.Context) (map[string]any, error)
	Count(ctx context.Context) (int, error)
	DeleteIndex(ctx context.Context) error
	// READ
	// this returns nil if not found, not exception
	GetById(ctx context.Context, id string, returnEmbedding bool) (*T, error, bool)
	GetByIds(ctx context.Context, ids []string, returnEmbedding bool) ([]T, error)
	// returns all points at once
	GetAllPoints(ctx context.Context, filter string, orderBy string, desc bool, returnEmbedding bool, allPoints *[]T) error
	FilterPoints(ctx context.Context, filter string, limit int, skip int, orderBy string, desc bool, returnEmbedding bool) (points []T, nextSkip int, totalCount int, err error)
	// WRITE
	UpsertPoint(ctx context.Context, point T) error
	UpsertPoints(ctx context.Context, points []T, batchSize int) error
	UpdateFields(ctx context.Context, id string, payload map[string]any) error
	DeletePoint(ctx context.Context, id string) error
	DeletePoints(ctx context.Context, filter string) error
	// SEARCH
	Suggest(ctx context.Context, query string, topK int, fuzzy bool, suggesterName string, searchFields []string, filter string, returnFields []string) ([]T, error)
	Autocomplete(ctx context.Context, query string, topK int, fuzzy bool, suggesterName string, searchFields []string, filter string) ([]string, error)
	SearchSemantic(ctx context.Context, query string, embedding []float32, searchVectorFields []string, searchVectorFieldWeights []float32, returnEmbedding bool, filter string, skip int, limit int) ([]T, int, error)
	SearchVector(ctx context.Context, embedding []float32, searchVectorFields []string, searchVectorFieldWeights []float32, returnEmbedding bool, filter string, limit int, threshold float32) ([]T, error)
	SearchText(ctx context.Context, query string, semantic bool, returnEmbedding bool, filter string, orderBy string, desc bool, skip int, limit int) ([]T, int, error)
}
