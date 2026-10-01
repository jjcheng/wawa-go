package feature_account_user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CreatePhoneNumberAblyToken struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (createPhoneNumberAblyToken *CreatePhoneNumberAblyToken) Validate() []exception.InputException {
	var errors []exception.InputException
	if createPhoneNumberAblyToken.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "missing phone number id"))
	}
	return errors
}

func (createPhoneNumberAblyToken CreatePhoneNumberAblyToken) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AblyTokenRequest] {
	if user == nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := createPhoneNumberAblyToken.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AblyTokenRequest](inputErrors)
	}
	if !helper.Any(user.AssignedPhoneNumbers, func(ap dto_account.AssignedPhoneNumber) bool {
		return ap.Id == createPhoneNumberAblyToken.PhoneNumberId
	}) {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	channelName := helper.GetPhoneNumberChannelName(createPhoneNumberAblyToken.PhoneNumberId)
	tokenRequest, err := dependencies.Ably.CreateConversationTokenRequest(channelName, fmt.Sprintf("user:%d", user.Id))
	if err != nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse(tokenRequest)
}

func (CreatePhoneNumberAblyToken) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create real time token in Ably for the phone number",
		"Creates a short-lived Ably token restricted to subscribe and presence on a phone number for notification.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/account/phone-numbers/:phone_number_id/token",
		true,
		true,
		types.APITagUser,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
		},
	)
}
