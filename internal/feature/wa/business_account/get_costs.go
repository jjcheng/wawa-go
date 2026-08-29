package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetCosts struct {
	MetaWABAId  string `form:"meta_waba_id" val:"required" description:"Meta WABA ID to query"`
	Start       int64  `form:"start" val:"required" description:"Unix timestamp for the analytics start"`
	End         int64  `form:"end" val:"required" description:"Unix timestamp for the analytics end"`
	Granularity string `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getCosts *GetCosts) Validate() []exception.InputException {
	getCosts.MetaWABAId = strings.TrimSpace(getCosts.MetaWABAId)
	getCosts.Granularity = strings.ToUpper(strings.TrimSpace(getCosts.Granularity))
	errors := []exception.InputException{}
	if getCosts.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	if getCosts.Start <= 0 {
		errors = append(errors, exception.NewInputException("start", "start must be a positive Unix timestamp"))
	}
	if getCosts.End <= getCosts.Start {
		errors = append(errors, exception.NewInputException("end", "end must be greater than start"))
	}
	if getCosts.Start > 0 && getCosts.Start < time.Now().UTC().AddDate(-1, 0, 0).Unix() {
		errors = append(errors, exception.NewInputException("start", "start must be within the last year"))
	}
	if getCosts.Granularity != "HALF_HOUR" && getCosts.Granularity != "DAILY" && getCosts.Granularity != "MONTHLY" {
		errors = append(errors, exception.NewInputException("granularity", "granularity must be HALF_HOUR, DAILY, or MONTHLY"))
	}
	return errors
}

func (getCosts GetCosts) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.PricingAnalytics] {
	if validationErrors := getCosts.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PricingAnalytics](validationErrors)
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, getCosts.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusNotFound, "business account not found")
		}
		return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessAccounts, err := dependencies.UnitOfWork.WAPhoneNumberRepository().ListBusinessAccounts(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if !helper.Any(businessAccounts, func(account dao_wa.BusinessAccount) bool {
		return account.MetaWABAId == businessAccount.MetaWABAId
	}) {
		return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusUnauthorized, "you are not authorized to view this")
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, businessAccount.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	costs, err := dependencies.Whatsapp.GetWABAPricingCosts(ctx, getCosts.MetaWABAId, getCosts.Start, getCosts.End, getCosts.Granularity, businessPortfolio.AccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, getCosts.MetaWABAId)
		return dto.NewFailedResponse[*dto_wa.PricingAnalytics](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(costs)
}

func (GetCosts) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp WABA pricing costs",
		"Gets WhatsApp pricing costs for a business account.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/wa/v1/business-accounts/costs",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to view this", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
