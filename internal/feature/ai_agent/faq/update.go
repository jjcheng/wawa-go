package feature_ai_agent_faq

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Update struct {
	Id      string `uri:"faq_id" val:"required" description:"id of the faq"`
	Enabled bool   `json:"enabled" description:"is it enabled"`
	Create
}

func (update *Update) Validate() []exception.InputException {
	errors := update.Create.Validate()
	update.Id = strings.TrimSpace(update.Id)
	if update.Id == "" {
		errors = append(errors, exception.NewInputException("id", "missing id"))
	}
	return errors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.Knowledge] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.Knowledge](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, update.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	repository := dependencies.UnitOfWork.AIKnowledgeRepository()
	knowledge, err := repository.Get(ctx, update.Id, false)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if knowledge == nil || knowledge.Category != types.AIKnowledgeCategoryFAQ {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusNotFound, "faq not found", nil)
	}
	// the faq must belong to the profile in the uri
	if knowledge.ProfileId != update.ProfileId {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	now := time.Now().UTC()
	fields := map[string]any{
		"title":           update.Question,
		"body":            update.Answer,
		"last_updated_at": now,
		"enabled":         update.Enabled,
	}
	// only re-embed what changed
	var textsToEmbed []string
	var embeddingFields []string
	if knowledge.Title != update.Question {
		textsToEmbed = append(textsToEmbed, update.Question)
		embeddingFields = append(embeddingFields, "title_embedding")
	}
	if knowledge.Body != update.Answer {
		textsToEmbed = append(textsToEmbed, update.Answer)
		embeddingFields = append(embeddingFields, "body_embedding")
	}
	if len(textsToEmbed) > 0 {
		embeddings, _, err := dependencies.LLM.Embed(ctx, textsToEmbed)
		if err != nil {
			return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
		}
		if len(embeddings) != len(textsToEmbed) {
			return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("expected %d embeddings, got %d", len(textsToEmbed), len(embeddings)))
		}
		for i, field := range embeddingFields {
			fields[field] = embeddings[i]
		}
	}
	if err := repository.UpdateFields(ctx, knowledge.Id, fields); err != nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	knowledge.Title = update.Question
	knowledge.Body = update.Answer
	knowledge.Enabled = update.Enabled
	knowledge.LastUpdatedAt = now
	return dto.NewSuccessResponse(knowledge)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent FAQ",
		"Updates the question and answer of an FAQ knowledge in Azure AI Search. Embeddings are regenerated only for the text that changed.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
		"/v1/ai-agent/profiles/:id/faqs/:faq_id",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("faq not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
