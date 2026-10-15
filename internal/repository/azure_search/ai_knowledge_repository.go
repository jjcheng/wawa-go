package azure_search

import (
	"context"
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type AIKnowledgeRepository struct {
	logger *service.Logger
	repository.SearchRepository[dto_ai.Knowledge]
}

func NewAIKnowledgeRepository(logger *service.Logger) repository.AIKnowledgeRepository {
	return &AIKnowledgeRepository{
		logger:           logger,
		SearchRepository: NewRepository[dto_ai.Knowledge](cfg.Default().AzureSearch.KnowledgesIndexName, cfg.Default().AzureSearch.KnowledgesSemanticConfigurationName, logger),
	}
}

// only search for type = 'text'
func (aiKnowledgeRepository *AIKnowledgeRepository) Search(ctx context.Context, query string, embedding []float32, profileIds []int32, topK int, threshold float32, useSemantic bool, vectorFiedsToSearch []string, vectorFieldWeights []float32) ([]dto_ai.Knowledge, error) {
	var filter string = fmt.Sprintf("enabled eq true and type eq '%s'", types.AIKnowledgeTypeText)
	if len(profileIds) > 0 {
		criteria := []string{}
		for _, profileId := range profileIds {
			criteria = append(criteria, fmt.Sprintf("profile_id eq %d", profileId))
		}
		filter += fmt.Sprintf(" and (%s)", strings.Join(criteria, " or "))
	}
	var knowledges []dto_ai.Knowledge
	var err error
	if useSemantic {
		knowledges, _, err = aiKnowledgeRepository.SearchSemantic(ctx, query, embedding, vectorFiedsToSearch, vectorFieldWeights, false, filter, 0, topK)
	} else {
		knowledges, err = aiKnowledgeRepository.SearchVector(ctx, embedding, vectorFiedsToSearch, vectorFieldWeights, false, filter, topK, threshold)
	}
	if err != nil {
		return nil, fmt.Errorf("error searching knowledges: %w", err)
	}
	return knowledges, nil
}

func (aiKnowledgeRepository *AIKnowledgeRepository) ListByProfileId(ctx context.Context, profileId int32, category types.AIKnowledgeCategory) ([]dto_ai.Knowledge, error) {
	filter := fmt.Sprintf("profile_id eq %d and category eq '%s'", profileId, category)
	allPoints := []dto_ai.Knowledge{}
	err := aiKnowledgeRepository.GetAllPoints(ctx, filter, "added_at", false, false, &allPoints)
	if err != nil {
		return nil, fmt.Errorf("error list knowledges by profile id: %d, error: %w", profileId, err)
	}
	return allPoints, nil
}

func (aiKnowledgeRepository *AIKnowledgeRepository) GetByBody(ctx context.Context, body string) (*dto_ai.Knowledge, error) {
	filter := fmt.Sprintf("enabled eq true and body eq '%s'", strings.ReplaceAll(body, "'", "''"))
	points, _, _, err := aiKnowledgeRepository.FilterPoints(ctx, filter, 1, 0, "added_at", false, true)
	if err != nil {
		return nil, fmt.Errorf("error getting knowledge by body: %w", err)
	}
	if len(points) == 0 {
		return nil, nil
	}
	return &points[0], nil
}

func (aiKnowledgeRepository *AIKnowledgeRepository) Get(ctx context.Context, id string, returnEmbedding bool) (*dto_ai.Knowledge, error) {
	filter := fmt.Sprintf("id eq '%s'", id)
	points, _, _, err := aiKnowledgeRepository.FilterPoints(ctx, filter, 1, 0, "added_at", false, returnEmbedding)
	if err != nil {
		return nil, fmt.Errorf("error get knowledge by id %s: %w", id, err)
	}
	if len(points) == 0 {
		return nil, nil
	}
	return &points[0], nil
}
