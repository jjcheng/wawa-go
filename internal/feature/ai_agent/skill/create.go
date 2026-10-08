package feature_ai_agent_skill

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
	ProfileId   int32  `uri:"id" val:"required" description:"id of the agent profile"`
	Title       string `json:"title" val:"required" description:"title of the skill"`
	Description string `json:"description" val:"required" description:"description of the skill"`
}

func (create *Create) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if create.ProfileId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	create.Title = strings.TrimSpace(create.Title)
	create.Description = strings.TrimSpace(create.Description)
	inputErrors = append(inputErrors, validateSkillContent(create.Title, create.Description)...)
	return inputErrors
}

func validateSkillContent(title string, description string) []exception.InputException {
	var inputErrors []exception.InputException
	if title == "" {
		inputErrors = append(inputErrors, exception.NewInputException("title", "title is required"))
	} else if len(title) > 100 {
		inputErrors = append(inputErrors, exception.NewInputException("title", "max length of title is 100"))
	}
	if description == "" {
		inputErrors = append(inputErrors, exception.NewInputException("description", "description is required"))
	} else if len(description) > 2000 {
		inputErrors = append(inputErrors, exception.NewInputException("description", "max length of description is 2000"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Skill] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Skill](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, create.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	skill := dao_ai_agent.Skill{
		ProfileId:   create.ProfileId,
		Title:       create.Title,
		Description: create.Description,
		Enabled:     true,
	}
	if err := dependencies.UnitOfWork.AIAgentSkillRepository().Insert(ctx, &skill); err != nil {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewSkill(skill)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create AI agent skill",
		"Creates an enabled skill for an AI agent profile in the business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/ai-agent/profiles/:id/skills",
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
