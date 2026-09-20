package feature_account_notification

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

type CreateAblyToken struct {
}

func (createAblyToken *CreateAblyToken) Validate() []exception.InputException {
	return nil
}

func (createAblyToken CreateAblyToken) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AblyTokenRequest] {
	if user == nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := createAblyToken.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AblyTokenRequest](inputErrors)
	}
	channelName := helper.GetNotificationChannelName(user.Id)
	tokenRequest, err := dependencies.Ably.CreateConversationTokenRequest(channelName, fmt.Sprintf("user:%d", user.Id))
	if err != nil {
		return dto.NewFailedResponse[*service.AblyTokenRequest](http.StatusServiceUnavailable, "realtime notification is unavailable")
	}
	return dto.NewSuccessResponse(tokenRequest)
}

func (CreateAblyToken) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp notification token",
		"Creates a short-lived Ably token restricted to subscribe and presence on site notification.",
		types.HttpRequestTypeNone,
		http.MethodPost,
		"/v1/account/notifications/token",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("realtime notification is unavailable", http.StatusServiceUnavailable)),
		},
	)
}
