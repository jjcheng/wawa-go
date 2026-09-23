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

type List struct {
	Type     types.NotificationType `form:"type" description:"type of the notifications to filter"`
	Read     *bool                  `form:"read" description:"true only return read, false only return unread"`
	Page     int                    `form:"page" description:"page number from 1"`
	PageSize int                    `form:"page_size" description:"page size, default 25"`
}

func (list *List) Validate() []exception.InputException {
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_account.Notification]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_account.Notification]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_account.Notification]](inputErrors)
	}
	notifications, totalCount, totalPages, err := dependencies.UnitOfWork.AccountNotificationRepository().ListByUserId(ctx, user.Id, list.Type, list.Read, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_account.Notification]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_account.Notification, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, dto_account.NewNotification(notification))
	}
	response := dto.NewPagedListResponse(items, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List notifications",
		"List notifications for the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/account/notifications",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
