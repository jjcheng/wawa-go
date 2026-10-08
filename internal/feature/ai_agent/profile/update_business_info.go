package feature_ai_agent_profile

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type UpdateBusinessInfo struct {
	Id    int32                           `uri:"id" val:"required" description:"id of the profile"`
	Items []dto_ai_agent.BusinessInfoItem `json:"items" val:"required" description:"business info items"`
}

func (update *UpdateBusinessInfo) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if update.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid profile id"))
	}
	if len(update.Items) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("items", "items are required"))
	} else {
		for i := range update.Items {
			if errors := update.Items[i].Validate(i); len(errors) > 0 {
				inputErrors = append(inputErrors, exception.NewInputException(fmt.Sprintf("items[%d].title", i), fmt.Sprintf("title is missing for item %d", i+1)))
			}
		}
	}
	return inputErrors
}

func (update UpdateBusinessInfo) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Profile] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Profile](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, update.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	profile.BusinessInfo = helper.Map(update.Items, func(i dto_ai_agent.BusinessInfoItem) map[string]any {
		return i.Payload()
	})
	if err := dependencies.UnitOfWork.AIAgentProfileRepository().UpdateBusinessInfo(ctx, profile); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewProfile(*profile)
	return dto.NewSuccessResponse(&result)
}

func (UpdateBusinessInfo) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent profile business info",
		"Replaces the business info of an AI agent profile without changing its other fields.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/ai-agent/profiles/:id/business-info",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
