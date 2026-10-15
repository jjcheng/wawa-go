package feature_ai_agent_faq

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

const (
	maxQuestionLength = 500
	maxAnswerLength   = 2000
)

type Create struct {
	ProfileId int32  `uri:"id" val:"required" description:"id of the agent profile"`
	Question  string `json:"question" val:"required" description:"the question, stored as the knowledge title"`
	Answer    string `json:"answer" val:"required" description:"the answer, stored as the knowledge body"`
}

func (create *Create) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if create.ProfileId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	create.Question = strings.TrimSpace(create.Question)
	create.Answer = strings.TrimSpace(create.Answer)
	if create.Question == "" {
		inputErrors = append(inputErrors, exception.NewInputException("question", "question is required"))
	} else if len(create.Question) > maxQuestionLength {
		inputErrors = append(inputErrors, exception.NewInputException("question", fmt.Sprintf("max length of question is %d", maxQuestionLength)))
	}
	if create.Answer == "" {
		inputErrors = append(inputErrors, exception.NewInputException("answer", "answer is required"))
	} else if len(create.Answer) > maxAnswerLength {
		inputErrors = append(inputErrors, exception.NewInputException("answer", fmt.Sprintf("max length of answer is %d", maxAnswerLength)))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.Knowledge] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.Knowledge](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, create.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	embeddings, tokens, err := dependencies.LLM.Embed(ctx, []string{create.Question, create.Answer})
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if len(embeddings) != 2 {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("expected 2 embeddings, got %d", len(embeddings)))
	}
	now := time.Now().UTC()
	knowledge := dto_ai.Knowledge{
		SearchResponseBase: dto.SearchResponseBase{
			Id:            uuid.NewString(),
			AddedAt:       now,
			LastUpdatedAt: now,
		},
		Type:           types.AIKnowledgeTypeText,
		Category:       types.AIKnowledgeCategoryFAQ,
		ProfileId:      create.ProfileId,
		Title:          create.Question,
		Body:           create.Answer,
		TitleEmbedding: embeddings[0],
		BodyEmbedding:  embeddings[1],
		Enabled:        true,
		Tokens:         tokens,
	}
	if err := dependencies.UnitOfWork.AIKnowledgeRepository().UpsertPoint(ctx, knowledge); err != nil {
		return dto.NewFailedResponse[*dto_ai.Knowledge](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// embeddings are large and not useful to the client
	knowledge.TitleEmbedding = nil
	knowledge.BodyEmbedding = nil
	return dto.NewSuccessResponse(&knowledge)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create AI agent FAQ",
		"Creates a text FAQ for an AI agent profile and stores it, with embeddings, as a knowledge in Azure AI Search.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/ai-agent/profiles/:id/faqs",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
