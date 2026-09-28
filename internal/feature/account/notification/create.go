package feature_account_notification

import (
	"context"
	"net/http"
	"strings"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Create struct {
	Category types.NotificationCategory `json:"category" val:"required" description:"category of the notification"`
	IconType types.NotificationIconType `json:"icon_type" description:"icon type of the notification"`
	Type     types.NotificationType     `json:"type" val:"required" description:"type of the notification"`
	Title    string                     `json:"title" val:"required" description:"title of the notification"`
	Body     string                     `json:"body" val:"body" description:"body of the notification"`
	URL      string                     `json:"url" description:"reference url if any"`
	ToUserId int32                      `json:"to_user_id" val:"required" description:"if of the user to send to"`
}

func (create *Create) Validate() []exception.InputException {
	create.Category = types.NotificationCategory(strings.TrimSpace(string(create.Category)))
	create.Type = types.NotificationType(strings.TrimSpace(string(create.Type)))
	create.IconType = types.NotificationIconType(strings.TrimSpace(string(create.IconType)))
	create.Title = strings.TrimSpace(create.Title)
	create.Body = strings.TrimSpace(create.Body)
	create.URL = strings.TrimSpace(create.URL)
	inputErrors := []exception.InputException{}
	if create.Type != types.NotificationTypeInfo && create.Type != types.NotificationTypeWarning && create.Type != types.NotificationTypeError && create.Type != types.NotificationTypeSuccess {
		inputErrors = append(inputErrors, exception.NewInputException("type", "invalid type"))
	}
	if create.Category != types.NotificationCategoryHandsOff && create.Category != types.NotificationCategoryPending {
		inputErrors = append(inputErrors, exception.NewInputException("category", "invalid category"))
	}
	if create.Title == "" {
		inputErrors = append(inputErrors, exception.NewInputException("title", "missing title"))
	}
	if create.Body == "" {
		inputErrors = append(inputErrors, exception.NewInputException("body", "missing body"))
	}
	if create.IconType == "" {
		inputErrors = append(inputErrors, exception.NewInputException("icon_type", "missing icon type"))
	}
	if create.ToUserId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("to_user_id", "missing to user id"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, dependencies *service.Dependencies) dto.Response[*dto_account.Notification] {
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.Notification](inputErrors)
	}
	notification := dao_account.Notification{
		UserId:   create.ToUserId,
		IconType: create.IconType,
		Title:    create.Title,
		Category: create.Category,
		Body:     create.Body,
		Type:     create.Type,
		URL:      create.URL,
	}
	if err := dependencies.UnitOfWork.AccountNotificationRepository().Insert(ctx, &notification); err != nil {
		return dto.NewFailedResponse[*dto_account.Notification](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_account.NewNotification(notification)
	channelName := helper.GetNotificationChannelName(create.ToUserId)
	if err := dependencies.Ably.Publish("notification", channelName, result); err != nil {
		dependencies.Logger.Error(err)
	}
	return dto.NewSuccessResponse(&result)
}

// not a public api
// func (Create) APISettings() feature.APISettings {
// 	return feature.NewAPISettings(
// 		"Create notification",
// 		"Create a notification for the authenticated user.",
// 		types.HttpRequestTypeJSON,
// 		http.MethodPost,
// 		"/v1/account/notifications",
// 		true,
// 		true,
// 		types.APITagAccount,
// 		[]feature.APIError{
// 			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
// 			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
// 		},
// 	)
// }
