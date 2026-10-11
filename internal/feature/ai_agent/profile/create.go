package feature_ai_agent_profile

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	Name        string `json:"name" val:"required" description:"name of the profile"`
	Description string `json:"description" val:"required" description:"description of the profile"`
}

func (create *Create) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	create.Name = strings.TrimSpace(create.Name)
	if create.Name == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "name is required"))
	} else if len(create.Name) > 50 {
		inputErrors = append(inputErrors, exception.NewInputException("name", "max length of name is 50"))
	}
	create.Description = strings.TrimSpace(create.Description)
	// if create.Description == "" {
	// 	inputErrors = append(inputErrors, exception.NewInputException("description", "description is required"))
	// }
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Profile] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Profile](inputErrors)
	}
	profile := dao_ai_agent.Profile{
		BusinessAccountId: user.BusinessAccountId,
		Name:              create.Name,
		Description:       create.Description,
	}
	if err := dependencies.UnitOfWork.AIAgentProfileRepository().Insert(ctx, &profile); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusConflict, "profile name is already in use", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewProfile(profile)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create AI agent profile",
		"Creates an AI agent profile with a name unique within the business account.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/ai-agent/profiles",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile name is already in use", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
