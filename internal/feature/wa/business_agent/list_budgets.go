package feature_wa_business_agent

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListBudgets struct {
}

func (listBudgets *ListBudgets) Validate() []exception.InputException {
	return nil
}

func (listBudgets ListBudgets) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]service.AgentBudget] {
	if user == nil {
		return dto.NewFailedResponse[[]service.AgentBudget](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]service.AgentBudget](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[[]service.AgentBudget](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := listBudgets.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]service.AgentBudget](inputErrors)
	}
	budgets, err := dependencies.Facebook.ListAgentBudgets(ctx, user.WA.BusinessPortfolio.MetaBusinessPortfolioId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[[]service.AgentBudget](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(budgets)
}

func (ListBudgets) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp business agent budgets",
		"Lists the configured budgets for a WhatsApp business agent.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/business-agent/budgets",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
