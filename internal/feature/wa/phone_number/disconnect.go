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

type Disconnect struct {
	Id int32 `uri:"id" description:"id of the phone number"`
}

func (deregister *Disconnect) Validate() []exception.InputException {
	if deregister.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid phone number id")}
	}
	return nil
}

func (deregister Disconnect) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not master")
	}
	if inputErrors := deregister.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, deregister.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil || phoneNumber.BusinessAccountId != user.WA.BusinessAccount.Id {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to deregister this phone number")
	}
	phoneDetails, err := dependencies.Whatsapp.GetPhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error())
	}
	if phoneDetails.IsOnBizApp {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "This number is co-existed on WhatsApp Business App, please re-register in the app instead.")
	}
	if phoneDetails.Status != "CONNECTED" {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "This number is not connected")
	}
	previousStatus := phoneNumber.Status
	phoneNumber.Status = types.WAPhoneNumberStatusDisconnected
	transaction := dependencies.UnitOfWork.BeginTransaction()
	if err := transaction.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		transaction.Rollback()
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := dependencies.Whatsapp.DisconnectPhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken); err != nil {
		phoneNumber.Status = previousStatus
		rollbackTransaction := dependencies.UnitOfWork.BeginTransaction()
		if rollbackErr := rollbackTransaction.WAPhoneNumberRepository().Update(ctx, phoneNumber); rollbackErr != nil {
			rollbackTransaction.Rollback()
			dependencies.Logger.ErrorFunction(rollbackErr, phoneNumber.Id)
		} else if rollbackErr := rollbackTransaction.CommitTransaction(); rollbackErr != nil {
			dependencies.Logger.ErrorFunction(rollbackErr, phoneNumber.Id)
		}
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error())
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Disconnect) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Deregister WhatsApp phone number",
		"Deregisters a phone number from WhatsApp Cloud API.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/wa/phone-numbers/:id/disconnect",
		true,
		false,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("This number is co-existed on WhatsApp Business App, please re-register in the app instead.", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
