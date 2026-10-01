package feature_broadcast

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	Id int32 `uri:"id" val:"required" description:"id of the broadcast"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid broadcast id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Broadcast] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Broadcast](errors)
	}
	broadcast, err := dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusNotFound, "broadcast not found", nil)
		}
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if broadcast.UserId != user.Id {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_customer.NewBroadcast(*broadcast)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a broadcast",
		"Get a broadcast belonging to the user.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/broadcasts/:id",
		true,
		true,
		types.APITagBroadcast,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("broadcast not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
