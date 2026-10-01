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

type Delete struct {
	Ids []int32 `json:"ids" val:"required" description:"ids of the notification to delete"`
}

func (delete *Delete) Validate() []exception.InputException {
	if len(delete.Ids) == 0 {
		return []exception.InputException{exception.NewInputException("ids", "missing ids")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := delete.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	notifications, err := dependencies.UnitOfWork.AccountNotificationRepository().ListByIds(ctx, delete.Ids, user.Id)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(notifications) != len(delete.Ids) {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "some notifications are no longer available", nil)
	}
	if err := dependencies.UnitOfWork.AccountNotificationRepository().DeleteByIds(ctx, delete.Ids, user.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete a notification",
		"Deletes a notification of the user.",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/v1/account/notifications",
		true,
		true,
		types.APITagNotification,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("notification not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
