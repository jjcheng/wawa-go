package feature_wa_message

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type CreateAblyToken struct {
	CustomerId int32 `form:"customer_id" val:"required" description:"id of the customer"`
}

func (createAblyToken *CreateAblyToken) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if createAblyToken.CustomerId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("customer_id", "missing customer ID"))
	}
	return inputErrors
}

func (createAblyToken CreateAblyToken) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AblyTokenRequest] {
	if user == nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || len(user.WA.PhoneNumbers) == 0 {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := createAblyToken.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AblyTokenRequest](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, createAblyToken.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusBadRequest, "customer not found", nil)
		}
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, customer.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusBadRequest, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	channelName := helper.GetChatChannelName(phoneNumber.MetaPhoneNumberId, customer.Token)
	tokenRequest, err := dependencies.Ably.CreateConversationTokenRequest(channelName, fmt.Sprintf("user:%d", user.Id))
	if err != nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusServiceUnavailable, "realtime chat is unavailable", err)
	}
	return dto.NewSuccessResponse(tokenRequest)
}

func (CreateAblyToken) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp conversation realtime chat token",
		"Creates a short-lived Ably token restricted to subscribe and presence on WhatsApp chat page.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/messages/chat-token",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("customer not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("realtime chat is unavailable", http.StatusServiceUnavailable)),
		},
	)
}
