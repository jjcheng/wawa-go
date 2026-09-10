package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetUsage struct {
	Start       int64                        `form:"start" val:"required" description:"Unix timestamp for the analytics start"`
	End         int64                        `form:"end" val:"required" description:"Unix timestamp for the analytics end"`
	Granularity types.WAAnalyticsGranularity `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getAnalytics *GetUsage) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if getAnalytics.Start <= 0 {
		errors = append(errors, exception.NewInputException("start", "start must be a positive Unix timestamp"))
	}
	if getAnalytics.End <= getAnalytics.Start {
		errors = append(errors, exception.NewInputException("end", "end must be greater than start"))
	}
	if getAnalytics.Start > 0 && getAnalytics.Start < time.Now().UTC().AddDate(-1, 0, 0).Unix() {
		errors = append(errors, exception.NewInputException("start", "start must be within the last year"))
	}
	if getAnalytics.Granularity != types.WAAnalyticsGranularityHalfHour && getAnalytics.Granularity != types.WAAnalyticsGranularityDay && getAnalytics.Granularity != types.WAAnalyticsGranularityMonth {
		errors = append(errors, exception.NewInputException("granularity", "granularity must be HALF_HOUR, DAY, or MONTH"))
	}
	return errors
}

func (getUsage GetUsage) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.MessageAnalytics] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusForbidden, "your are not authenticated")
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, "your are not master")
	}
	if errors := getUsage.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.MessageAnalytics](errors)
	}
	businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, "you are not authorized to view this")
		}
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	usage, err := dependencies.Whatsapp.GetWABAUsage(ctx, businessAccount.MetaWABAId, getUsage.Start, getUsage.End, getUsage.Granularity, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusBadGateway, err.Error())
	}
	return dto.NewSuccessResponse(usage)
}

func (GetUsage) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business account usage",
		"Gets message delivery usage for a WhatsApp Business Account.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/business-accounts/usage",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to view this", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
