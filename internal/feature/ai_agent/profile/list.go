package feature_ai_agent_profile

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
}

func (list *List) Validate() []exception.InputException {
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai_agent.Profile] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_ai_agent.Profile](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]dto_ai_agent.Profile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai_agent.Profile](inputErrors)
	}
	profiles, err := dependencies.UnitOfWork.AIAgentProfileRepository().ListByBusinessAccountId(ctx, user.BusinessAccountId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_ai_agent.Profile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, dto_ai_agent.NewProfile(profile))
	}
	return dto.NewSuccessResponse(items)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List AI agent profiles",
		"Lists AI agent profiles in the business account.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/ai-agent/profiles",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
