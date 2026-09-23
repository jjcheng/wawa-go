package feature_wa_message

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	Id int32 `uri:"id" val:"required" description:"id of the message"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid message id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Message] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Message](inputErrors)
	}
	message, err := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Message](http.StatusNotFound, "message not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Message](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if phoneNumber.UserId != user.Id {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_wa.NewMessage(*message)
	// get statuses
	statuses, err := dependencies.UnitOfWork.WAMessageStatusRepository().ListByMessageId(ctx, message.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var statusList []dto_wa.MessageStatus
	for _, status := range statuses {
		statusList = append(statusList, dto_wa.NewMessageStatus(status))
	}
	result.Statuses = statusList
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp message",
		"Gets a WhatsApp message by its local ID.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/messages/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("not your message", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("message not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
