package feature_wa_message

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type StartTyping struct {
	MessageId int32 `form:"message_id" val:"required" description:"id of the replying message"`
}

func (startTyping *StartTyping) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if startTyping.MessageId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("message_id", "missing message id"))
	}
	return inputErrors
}

func (startTyping StartTyping) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := startTyping.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	message, err := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, startTyping.MessageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "message not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == message.PhoneNumberId
	}) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if strings.TrimSpace(message.WAMessageId) == "" {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "message has no WhatsApp message id", nil)
	}
	phoneNumber := helper.First(user.WA.PhoneNumbers, func(phoneNumber dto_wa.PhoneNumber) bool {
		return phoneNumber.Id == message.PhoneNumberId
	})
	if phoneNumber == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if _, err := dependencies.Whatsapp.StartTyping(ctx, phoneNumber.MetaPhoneNumberId, message.WAMessageId, user.WA.BusinessPortfolioAccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (StartTyping) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Start WhatsApp typing indicator",
		"Start the typing indicator for a WhatsApp message by the user.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/messages/start-typing",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("message not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
