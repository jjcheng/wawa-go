package feature_wa_phone_number

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

type GetUsage struct {
	MetaWABAId  string `form:"meta_waba_id" val:"required" description:"Meta WABA ID"`
	Start       int64  `form:"start" val:"required" description:"Unix timestamp for the analytics start"`
	End         int64  `form:"end" val:"required" description:"Unix timestamp for the analytics end"`
	Granularity string `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getUsage *GetUsage) Validate() []exception.InputException {
	getUsage.MetaWABAId = strings.TrimSpace(getUsage.MetaWABAId)
	getUsage.Granularity = strings.ToUpper(strings.TrimSpace(getUsage.Granularity))
	errors := []exception.InputException{}
	if getUsage.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	if getUsage.Start <= 0 {
		errors = append(errors, exception.NewInputException("start", "start must be a positive Unix timestamp"))
	}
	if getUsage.End <= getUsage.Start {
		errors = append(errors, exception.NewInputException("end", "end must be greater than start"))
	}
	if getUsage.Start > 0 && getUsage.Start < time.Now().UTC().AddDate(-1, 0, 0).Unix() {
		errors = append(errors, exception.NewInputException("start", "start must be within the last year"))
	}
	if getUsage.Granularity != "HALF_HOUR" && getUsage.Granularity != "DAY" && getUsage.Granularity != "MONTH" {
		errors = append(errors, exception.NewInputException("granularity", "granularity must be HALF_HOUR, DAY, or MONTH"))
	}
	return errors
}

func (getUsage GetUsage) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.PhoneNumberMessageAnalytics] {
	if validationErrors := getUsage.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.PhoneNumberMessageAnalytics](validationErrors)
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, getUsage.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusNotFound, "business account not found")
		}
		return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessAccounts, err := dependencies.UnitOfWork.WAPhoneNumberRepository().ListBusinessAccounts(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if !helper.Any(businessAccounts, func(account dao_wa.BusinessAccount) bool {
		return account.MetaWABAId == businessAccount.MetaWABAId
	}) {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusUnauthorized, "you are not authorized to view this")
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, businessAccount.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	analytics, err := dependencies.Whatsapp.GetPhoneNumberUsage(ctx, getUsage.MetaWABAId, getUsage.Start, getUsage.End, getUsage.Granularity, businessPortfolio.AccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, getUsage.MetaWABAId)
		return dto.NewFailedResponse[[]dto_wa.PhoneNumberMessageAnalytics](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(analytics)
}

func (GetUsage) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp phone number usage",
		"Gets message delivery usage for phone numbers in a WhatsApp Business Account.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/wa/v1/phone-numbers/usage",
		true,
		false,
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
