package feture_ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Search struct {
	Query     string
	ProfileId int32
	TopK      int
	Semantic  bool
	Threshold float32
}

func (search *Search) Validate() []exception.InputException {
	var errors []exception.InputException
	search.Query = strings.TrimSpace(search.Query)
	if search.Query == "" {
		errors = append(errors, exception.NewInputException("query", "missing query"))
	}
	if search.ProfileId <= 0 {
		errors = append(errors, exception.NewInputException("profile_id", "missing profile id"))
	}
	if search.TopK <= 0 {
		search.TopK = 5
	}
	if search.Threshold <= 0 {
		search.Threshold = 0.5
	}
	return errors
}

// Handle searches the text knowledges of the profile in Azure AI Search.
func (search *Search) Handle(ctx context.Context, dependencies *service.Dependencies) dto.Response[[]dto_ai.Knowledge] {
	if errors := search.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai.Knowledge](errors)
	}
	embeddings, _, err := dependencies.LLM.Embed(ctx, []string{search.Query})
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if len(embeddings) != 1 {
		return dto.NewFailedResponse[[]dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("expected 1 embedding, got %d", len(embeddings)))
	}
	knowledges, err := dependencies.UnitOfWork.AIKnowledgeRepository().Search(ctx, search.Query, embeddings[0], []int32{search.ProfileId}, search.TopK, search.Threshold, search.Semantic, []string{"title_embedding", "body_embedding"}, []float32{1, 1})
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse(knowledges)
}
