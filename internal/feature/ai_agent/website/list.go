package feature_ai_agent_website

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

type List struct {
	ProfileId int32 `uri:"id" val:"required" description:"id of the agent profile"`
}

func (list *List) Validate() []exception.InputException {
	if list.ProfileId <= 0 {
		return []exception.InputException{exception.NewInputException("profile_id", "invalid profile id")}
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai_agent.Website] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai_agent.Website](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, list.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	websites, err := dependencies.UnitOfWork.AIAgentWebsiteRepository().ListByProfileId(ctx, profile.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai_agent.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_ai_agent.Website, 0, len(websites))
	for _, website := range websites {
		items = append(items, dto_ai_agent.NewWebsite(website))
	}
	return dto.NewSuccessResponse(items)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List AI agent websites",
		"Lists websites for an AI agent profile in the business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/ai-agent/profiles/:id/websites",
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
