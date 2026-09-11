package feature_account_user

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetDashboard struct {
	BusinessAccount bool `form:"business_account" description:"only MASTER can set this to true"`
}

type Dashboard struct {
	ActivePhoneNumbers          int `json:"active_phone_numbers"`
	ActiveCustomers             int `json:"active_customers"`
	MessagesSentLast30Days      int `json:"messages_sent_last_30_days"`
	MessagesDeliveredLast30Days int `json:"messages_delivered_last_30_days"`
}

func (GetDashboard) Validate() []exception.InputException {
	return nil
}

func (getDashboard GetDashboard) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*Dashboard] {
	if user == nil {
		return dto.NewFailedResponse[*Dashboard](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := getDashboard.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*Dashboard](inputErrors)
	}
	var activePhoneNumbers, activeCustomers, messagesSent, messagesDelivered int
	start := time.Now().UTC().AddDate(0, 0, -30).Unix()
	end := time.Now().UTC().Unix()
	if getDashboard.BusinessAccount && user.Type == types.UserTypeMaster {
		businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		// get phone numbers by WABA
		activePhoneNumbers, err = dependencies.UnitOfWork.WAPhoneNumberRepository().CountByMetaBusinessAccountId(ctx, businessAccount.MetaWABAId)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		// get active customer count by WABA
		activeCustomers, err = dependencies.UnitOfWork.CustomerRepository().CountActiveByMetaBusinessAccountId(ctx, businessAccount.MetaWABAId)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		usage, err := dependencies.Whatsapp.GetWABAUsage(ctx, businessAccount.MetaWABAId, start, end, types.WAAnalyticsGranularityDay, businessPortfolio.AccessToken)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusBadGateway, err.Error())
		}
		messagesSent = usage.TotalSent
		messagesDelivered = usage.TotalDelivered
	} else {
		// get user's phone number
		phoneNumber, _, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx, user.Id)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
			}
		}
		if phoneNumber != nil {
			if phoneNumber.Status == types.WAPhoneNumberStatusConnected {
				activePhoneNumbers = 1
			}
			businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
			if err != nil {
				return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
			}
			usage, err := dependencies.Whatsapp.GetPhoneNumberUsage(ctx, businessAccount.MetaWABAId, []string{phoneNumber.WAId}, start, end, types.WAAnalyticsGranularityDay, businessPortfolio.AccessToken)
			if err != nil {
				return dto.NewFailedResponse[*Dashboard](http.StatusBadGateway, err.Error())
			}
			messagesSent = usage.TotalSent
			messagesDelivered = usage.TotalDelivered
		}
		// get user's customers
		ac, err := dependencies.UnitOfWork.CustomerRepository().CountActiveByUserId(ctx, user.Id)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		activeCustomers = ac
	}
	return dto.NewSuccessResponse(&Dashboard{
		ActivePhoneNumbers:          activePhoneNumbers,
		ActiveCustomers:             activeCustomers,
		MessagesSentLast30Days:      messagesSent,
		MessagesDeliveredLast30Days: messagesDelivered,
	})
}

func (GetDashboard) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get dashboard",
		"Gets account dashboard counts for phone numbers, customers, and recent outgoing messages.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/account/users/me/dashboard",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
