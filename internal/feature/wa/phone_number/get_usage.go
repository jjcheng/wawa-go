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
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetUsage struct {
	Id          int32                        `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	Start       int64                        `form:"start" val:"required" description:"Unix timestamp for the analytics start"`
	End         int64                        `form:"end" val:"required" description:"Unix timestamp for the analytics end"`
	Granularity types.WAAnalyticsGranularity `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getUsage *GetUsage) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if getUsage.Id <= 0 {
		errors = append(errors, exception.NewInputException("id", "missing ID"))
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
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		// in case user supplied WAIds which don't belong to him, use this to filter
		if user.WA == nil {
			return dto.NewSuccessResponse(&dto_wa.MessageAnalytics{})
		}
		// make sure the id belong to the user
		if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == getUsage.Id
		}) {
			return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	}
	if user.WA == nil || user.WA.BusinessAccount == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if validationErrors := getUsage.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.MessageAnalytics](validationErrors)
	}
	// get waIds by ids
	var waId string
	if user.Type == types.UserTypeMaster {
		phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getUsage.Id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusNotFound, "phone number not found", nil)
			}
			return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if phoneNumber.BusinessAccountId != user.BusinessAccountId {
			return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
		waId = phoneNumber.WAId
	} else {
		phoneNumber := helper.First(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == getUsage.Id
		})
		// no need to check nil again, it's checked earlier
		waId = phoneNumber.WAId
	}
	analytics, err := dependencies.Whatsapp.GetPhoneNumberUsage(ctx, user.WA.BusinessAccount.WABAId, []string{waId}, getUsage.Start, getUsage.End, getUsage.Granularity, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.MessageAnalytics](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(analytics)
}

func (GetUsage) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp phone number usage",
		"Get message delivery usage for phone numbers by WhatsApp API. Only MASTER user can access this endpoint.",
		types.HttpRequestTypeUriQuery,
		http.MethodGet,
		"/v1/wa/phone-numbers/:phone_number_id/usage",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to view this", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
