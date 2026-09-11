package feature_wa_message

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CreateToken struct {
	PhoneNumberID      string `json:"phone_number_id" val:"required" description:"Meta ID of the business phone number"`
	CustomerWAId       string `json:"customer_wa_id" description:"Customer WhatsApp ID" example:"6590073708"`
	CustomerMetaUserID string `json:"customer_meta_user_id" description:"Customer Meta business-scoped user ID"`
}

func (createToken *CreateToken) Validate() []exception.InputException {
	createToken.PhoneNumberID = strings.TrimSpace(createToken.PhoneNumberID)
	createToken.CustomerWAId = strings.TrimSpace(createToken.CustomerWAId)
	createToken.CustomerMetaUserID = strings.TrimSpace(createToken.CustomerMetaUserID)
	inputErrors := []exception.InputException{}
	if createToken.PhoneNumberID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "missing phone number ID"))
	}
	if createToken.CustomerWAId == "" && createToken.CustomerMetaUserID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("customer_wa_id", "customer WA Id or Meta user ID is required"))
	}
	return inputErrors
}

func (createToken CreateToken) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AblyTokenRequest] {
	if user == nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := createToken.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AblyTokenRequest](inputErrors)
	}
	if user.WA == nil || user.WA.PhoneNumber_ == nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	if user.WA.PhoneNumber_.MetaPhoneNumberId != createToken.PhoneNumberID {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	channelName := helper.GetChatChannelName(createToken.PhoneNumberID, createToken.CustomerWAId, createToken.CustomerMetaUserID)
	tokenRequest, err := dependencies.Ably.CreateConversationTokenRequest(channelName, fmt.Sprintf("user:%d", user.Id))
	if err != nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusServiceUnavailable, "realtime messaging is unavailable")
	}
	return dto.NewSuccessResponse(tokenRequest)
}

func (CreateToken) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp conversation realtime token",
		"Creates a short-lived Ably token restricted to subscribe and presence on one WhatsApp conversation channel.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/messages/realtime-token",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("customer not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("realtime messaging is unavailable", http.StatusServiceUnavailable)),
		},
	)
}
