package feature_broadcast

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Cancel struct {
	Id int32 `uri:"id" val:"required" description:"id of the broadcast to cancel"`
}

func (cancel *Cancel) Validate() []exception.InputException {
	if cancel.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid broadcast id")}
	}
	return nil
}

func (cancel Cancel) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if errors := cancel.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	broadcast, err := dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, cancel.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "broadcast not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if broadcast.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusNotFound, "broadcast not found")
	}
	if broadcast.Status != types.BroadcastStatusPending {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "only pending broadcasts can be cancelled")
	}
	// start a transaction to also cancel all broadcast_recipients
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	if err := transaction.BroadcastRepository().UpdateFields(ctx, broadcast.Id, map[string]any{
		"status": types.BroadcastStatusCancelled,
	}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Cancel) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Cancel WhatsApp broadcast",
		"Cancels a pending broadcast belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodPatch,
		"/v1/broadcasts/:id/cancel",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("broadcast not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("only pending broadcasts can be cancelled", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
