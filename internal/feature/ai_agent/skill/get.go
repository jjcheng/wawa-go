package feature_ai_agent_skill

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	ProfileId int32 `uri:"id" val:"required" description:"id of the agent profile"`
	Id        int32 `uri:"skill_id" val:"required" description:"id of the skill"`
}

func (get *Get) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if get.ProfileId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	if get.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("skill_id", "invalid skill id"))
	}
	return inputErrors
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Skill] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Skill](inputErrors)
	}
	skill, err := dependencies.UnitOfWork.AIAgentSkillRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusNotFound, "skill not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if skill.ProfileId != get.ProfileId {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, skill.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_ai_agent.NewSkill(*skill)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get AI agent skill",
		"Gets an AI agent skill belonging to the profile in the business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/ai-agent/profiles/:id/skills/:skill_id",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("skill not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
