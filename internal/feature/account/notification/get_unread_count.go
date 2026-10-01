package feature_account_notification

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type GetUnreadCount struct {
}

func (getUnreadCount *GetUnreadCount) Validate() []exception.InputException {
	return nil
}

func (getUnreadCount GetUnreadCount) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[int] {
	if user == nil {
		return dto.NewFailedResponse[int](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := getUnreadCount.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[int](inputErrors)
	}
	count, err := dependencies.UnitOfWork.AccountNotificationRepository().GetUnreadCount(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[int](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse(count)
}

func (GetUnreadCount) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get unread notification count",
		"Gets the number of unread notifications for the user.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/account/notifications/unread-count",
		true,
		true,
		types.APITagNotification,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
