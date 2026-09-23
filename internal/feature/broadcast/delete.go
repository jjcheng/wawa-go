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

type Delete struct {
	Id int32 `uri:"id" val:"required" description:"id of the broadcast to delete"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid broadcast id")}
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
	broadcast, err := dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, delete.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "broadcast not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if broadcast.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if broadcast.Status != types.BroadcastStatusCancelled {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "only cancelled broadcasts can be deleted", nil)
	}
	// delete any attachementurl
	if broadcast.AttachmentURL != "" {
		dependencies.File.DeleteFile(broadcast.AttachmentURL)
	}
	if err := dependencies.UnitOfWork.BroadcastRepository().DeleteById(ctx, broadcast.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp broadcast",
		"Deletes a cancelled broadcast belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/broadcasts/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("broadcast not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("only cancelled broadcasts can be deleted", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
