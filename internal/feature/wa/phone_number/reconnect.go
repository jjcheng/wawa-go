package feature_wa_phone_number

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

type Reconnect struct {
	Id int32 `uri:"id" description:"id of the phone number"`
}

func (reconnect *Reconnect) Validate() []exception.InputException {
	if reconnect.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid phone number id")}
	}
	return nil
}

func (reconnect Reconnect) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := reconnect.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, reconnect.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil || phoneNumber.BusinessAccountId != user.WA.BusinessAccount.Id {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	phoneDetails, err := dependencies.Whatsapp.GetPhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	if phoneDetails.Status != "DISCONNECTED" {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "phone number is not disconnected", nil)
	}
	if phoneNumber.RegistrationPin == "" {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "registration PIN is missing", nil)
	}
	if err := dependencies.Whatsapp.ReconnectPhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, phoneNumber.RegistrationPin, user.WA.BusinessPortfolioAccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	previousStatus := phoneNumber.Status
	phoneNumber.Status = types.WAPhoneNumberStatusConnected
	transaction := dependencies.UnitOfWork.BeginTransaction()
	if err := transaction.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		transaction.Rollback()
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if err := transaction.CommitTransaction(); err != nil {
		phoneNumber.Status = previousStatus
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Reconnect) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Reconnect WhatsApp phone number",
		"Registers a disconnected phone number again with WhatsApp Cloud API.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/wa/phone-numbers/:id/reconnect",
		true,
		false,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("phone number is not disconnected", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("registration PIN is missing", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
