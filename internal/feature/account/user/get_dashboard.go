package feature_account_user

import (
	"context"
	"net/http"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
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
	if user.WA == nil || user.WA.BusinessAccount == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*Dashboard](http.StatusUnauthorized, "you are not authorized")
	}
	if inputErrors := getDashboard.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*Dashboard](inputErrors)
	}
	var activePhoneNumbers, activeCustomers, messagesSent, messagesDelivered int
	start := time.Now().UTC().AddDate(0, 0, -30).Unix()
	end := time.Now().UTC().Unix()
	var err error
	if getDashboard.BusinessAccount && user.Type == types.UserTypeMaster {
		// get phone numbers by WABA
		activePhoneNumbers, err = dependencies.UnitOfWork.WAPhoneNumberRepository().CountByBusinessAccountId(ctx, user.WA.BusinessAccount.Id)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		// get active customer count by WABA
		activeCustomers, err = dependencies.UnitOfWork.CustomerRepository().CountActiveByBusinessAccountId(ctx, user.WA.BusinessAccount.Id)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		usage, err := dependencies.Whatsapp.GetWABAUsage(ctx, user.WA.BusinessAccount.WABAId, start, end, types.WAAnalyticsGranularityDay, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusBadGateway, err.Error())
		}
		messagesSent = usage.TotalSent
		messagesDelivered = usage.TotalDelivered
	} else {
		if user.WA.PhoneNumber_ != nil && user.WA.PhoneNumber_.Status == types.WAPhoneNumberStatusConnected {
			activePhoneNumbers = 1
		}
		usage, err := dependencies.Whatsapp.GetPhoneNumberUsage(ctx, user.WA.BusinessAccount.WABAId, []string{user.WA.PhoneNumber_.WAId}, start, end, types.WAAnalyticsGranularityDay, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			return dto.NewFailedResponse[*Dashboard](http.StatusBadGateway, err.Error())
		}
		messagesSent = usage.TotalSent
		messagesDelivered = usage.TotalDelivered
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
