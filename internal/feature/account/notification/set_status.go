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

type SetStatus struct {
	Ids  []int32 `json:"ids" val:"required" description:"id of the notification"`
	Read bool    `json:"read" description:"has read or not"`
}

func (setStatus *SetStatus) Validate() []exception.InputException {
	var errors []exception.InputException
	if len(setStatus.Ids) == 0 {
		errors = append(errors, exception.NewInputException("ids", "missing notification ids"))
	}
	return errors
}

func (setStatus SetStatus) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := setStatus.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	notifications, err := dependencies.UnitOfWork.AccountNotificationRepository().ListByIds(ctx, setStatus.Ids, user.Id)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(notifications) != len(setStatus.Ids) {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "some notifications are no longer available", nil)
	}
	err = dependencies.UnitOfWork.AccountNotificationRepository().SetStatusByIds(ctx, user.Id, setStatus.Ids, setStatus.Read)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (SetStatus) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Set notification read status",
		"Marks a notification as read or unread for the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPatch,
		"/v1/account/notifications/status",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("notification not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
