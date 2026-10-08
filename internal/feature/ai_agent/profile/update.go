package feature_ai_agent_profile

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Update struct {
	Id          int32  `uri:"id" val:"required" description:"id of the profile"`
	Name        string `json:"name" val:"required" description:"name of the profile"`
	Description string `json:"description" val:"required" description:"description of the profile"`
}

func (update *Update) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if update.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid profile id"))
	}
	update.Name = strings.TrimSpace(update.Name)
	if update.Name == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "name is required"))
	} else if len(update.Name) > 50 {
		inputErrors = append(inputErrors, exception.NewInputException("name", "max length of name is 50"))
	}
	update.Description = strings.TrimSpace(update.Description)
	if update.Description == "" {
		inputErrors = append(inputErrors, exception.NewInputException("description", "description is required"))
	}
	return inputErrors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Profile] {
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
	profile.Name = update.Name
	profile.Description = update.Description
	fields := map[string]any{
		"name":        profile.Name,
		"description": profile.Description,
	}
	if err := dependencies.UnitOfWork.AIAgentProfileRepository().UpdateFields(ctx, profile.Id, fields); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusConflict, "profile name is already in use", nil)
		}
		if updatedAt, ok := fields["last_updated_at"].(time.Time); ok {
			profile.LastUpdatedAt = updatedAt
		}
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewProfile(*profile)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent profile",
		"Updates an AI agent profile with a name unique within the business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
		"/v1/ai-agent/profiles/:id",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("profile name is already in use", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
