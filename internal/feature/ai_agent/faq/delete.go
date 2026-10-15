package feature_ai_agent_faq

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Delete struct {
	ProfileId int32  `uri:"id" val:"required" description:"id of the agent profile"`
	Id        string `uri:"faq_id" val:"required" description:"id of the faq"`
}

func (delete *Delete) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if delete.ProfileId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	delete.Id = strings.TrimSpace(delete.Id)
	// the id is used in an Azure Search filter, so it must be a UUID
	if _, err := uuid.Parse(delete.Id); err != nil {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid id"))
	}
	return inputErrors
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := delete.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, delete.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	repository := dependencies.UnitOfWork.AIKnowledgeRepository()
	knowledge, err := repository.Get(ctx, delete.Id, false)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if knowledge == nil || knowledge.Category != types.AIKnowledgeCategoryFAQ {
		return dto.NewFailedResponse[any](http.StatusNotFound, "faq not found", nil)
	}
	// the faq must belong to the profile in the uri
	if knowledge.ProfileId != delete.ProfileId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := repository.DeletePoint(ctx, delete.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete AI agent FAQ",
		"Deletes an FAQ knowledge of an AI agent profile from Azure AI Search.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/ai-agent/profiles/:id/faqs/:faq_id",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("faq not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
