package feature_ai_agent_profile

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

type UpdateBudgets struct {
	Id             int32 `uri:"id" val:"required" description:"id of the profile"`
	BudgetDaily    int32 `json:"budget_daily" val:"required" description:"daily budget * 1000"`
	Budget7Days    int32 `json:"budget_7_days" val:"required" description:"7 day budget * 1000"`
	Budget30Days   int32 `json:"budget_30_days" val:"required" description:"30 day budget * 1000"`
	UTCOffsetHours int32 `json:"utc_offset_hours" val:"required" description:"offset hours from UTC to reset budget"`
}

func (update *UpdateBudgets) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if update.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid profile id"))
	}
	if update.BudgetDaily < 1000 {
		inputErrors = append(inputErrors, exception.NewInputException("budget_daily", "daily budget must be at least 1"))
	}
	if update.Budget7Days < update.BudgetDaily {
		inputErrors = append(inputErrors, exception.NewInputException("budget_7_days", "7 day budget must be at least same as daily budget"))
	}
	if update.Budget30Days > 0 && update.Budget30Days < update.Budget7Days {
		inputErrors = append(inputErrors, exception.NewInputException("budget_30_days", "30 day budget must be at least same as 7 day budget"))
	}
	return inputErrors
}

func (update UpdateBudgets) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Profile] {
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
	profile.BudgetDaily = update.BudgetDaily
	profile.Budget7Days = update.Budget7Days
	profile.Budget30Days = update.Budget30Days
	profile.UTCOffsetHours = update.UTCOffsetHours
	if err := dependencies.UnitOfWork.AIAgentProfileRepository().Update(ctx, profile); err != nil {
		return dto.NewFailedResponse[*dto_ai_agent.Profile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewProfile(*profile)
	return dto.NewSuccessResponse(&result)
}

func (UpdateBudgets) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent profile budgets",
		"Updates profile budgets: daily must be at least 1000, 7 day must be at least daily, and 30 day must be at least 7 day.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/ai-agent/profiles/:id/budgets",
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
