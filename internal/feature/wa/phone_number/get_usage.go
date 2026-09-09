package feature_wa_phone_number

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
	WAIds       []string                     `form:"wa_ids" val:"required" description:"up to 10 wa Ids"`
	Start       int64                        `form:"start" val:"required" description:"Unix timestamp for the analytics start"`
	End         int64                        `form:"end" val:"required" description:"Unix timestamp for the analytics end"`
	Granularity types.WAAnalyticsGranularity `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getUsage *GetUsage) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if len(getUsage.WAIds) == 0 {
		errors = append(errors, exception.NewInputException("wa_ids", "missing WA IDs"))
	}
	if len(getUsage.WAIds) > 10 {
		errors = append(errors, exception.NewInputException("wa_ids", "at most 10 WA IDs are allowed"))
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
	if getUsage.Granularity != types.WAAnalyticsGranularityHalfHour && getUsage.Granularity != types.WAAnalyticsGranularityDay && getUsage.Granularity != types.WAAnalyticsGranularityMonth {
		errors = append(errors, exception.NewInputException("granularity", "granularity must be HALF_HOUR, DAY, or MONTH"))
	}
	return errors
}

func (getUsage GetUsage) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.MessageAnalytics] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusForbidden, "you are not authenticated")
	}
	if validationErrors := getUsage.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.MessageAnalytics](validationErrors)
	}
	businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, "you are not authorized to view this")
		}
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	analytics, err := dependencies.Whatsapp.GetPhoneNumberUsage(ctx, businessAccount.MetaWABAId, getUsage.WAIds, getUsage.Start, getUsage.End, getUsage.Granularity, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusBadGateway, err.Error())
	}
	return dto.NewSuccessResponse(analytics)
}

func (GetUsage) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp phone number usage",
		"Gets message delivery usage for phone numbers in a WhatsApp Business Account.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/phone-numbers/usage",
		true,
		false,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to view this", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
