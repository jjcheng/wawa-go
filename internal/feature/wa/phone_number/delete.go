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

type Delete struct {
	Id int32 `uri:"id" val:"required" description:"id of the phone number"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid phone number id")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	//return dto.NewFailedResponse[any](http.StatusNotImplemented, "not in use")
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not master")
	}
	// only can delete if there is no message, no message event, no customer, no campaign
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, delete.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if user.WA == nil || user.WA.PhoneNumber_ == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized")
	}
	if user.WA.BusinessAccount.Id != phoneNumber.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found")
	}
	// TODO: remove all messages
	if err := dependencies.Whatsapp.RemovePhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolio.MetaBusinessPortfolioId); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error())
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().DeleteById(ctx, phoneNumber.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Remove WhatsApp phone number",
		"Removes an assigned WhatsApp phone number from WhatsApp and the local account.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/wa/phone-numbers/:id",
		true,
		false,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("Meta business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
