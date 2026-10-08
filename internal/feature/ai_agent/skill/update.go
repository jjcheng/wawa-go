package feature_ai_agent_skill

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
	ProfileId   int32  `uri:"id" val:"required" description:"id of the agent profile"`
	Id          int32  `uri:"skill_id" val:"required" description:"id of the skill"`
	Title       string `json:"title" val:"required" description:"title of the skill"`
	Description string `json:"description" val:"required" description:"description of the skill"`
	Enabled     bool   `json:"enabled" val:"required" description:"if the skill is enabled"`
}

func (update *Update) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if update.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid skill id"))
	}
	update.Title = strings.TrimSpace(update.Title)
	update.Description = strings.TrimSpace(update.Description)
	inputErrors = append(inputErrors, validateSkillContent(update.Title, update.Description)...)
	return inputErrors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Skill] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Skill](inputErrors)
	}
	skill, err := dependencies.UnitOfWork.AIAgentSkillRepository().GetById(ctx, update.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusNotFound, "skill not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if skill.ProfileId != update.ProfileId {
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
	fields := map[string]any{"title": update.Title, "description": update.Description}
	if err := dependencies.UnitOfWork.AIAgentSkillRepository().UpdateFields(ctx, skill.Id, fields); err != nil {
		return dto.NewFailedResponse[*dto_ai_agent.Skill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	skill.Title = update.Title
	skill.Description = update.Description
	skill.Enabled = update.Enabled
	if updatedAt, ok := fields["last_updated_at"].(time.Time); ok {
		skill.LastUpdatedAt = updatedAt
	}
	result := dto_ai_agent.NewSkill(*skill)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent skill",
		"Updates the title and description of an AI agent skill in the business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
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
