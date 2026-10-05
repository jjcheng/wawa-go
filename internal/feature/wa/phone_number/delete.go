package feature_wa_phone_number

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Delete struct {
	Id int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.Id <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, delete.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// just check they are under 1 business account
	if phoneNumber == nil || phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	phoneNumberDetails, err := dependencies.Whatsapp.GetPhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		if !strings.Contains(err.Error(), "does not exist") && !strings.Contains(err.Error(), "has been deleted") {
			return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
		}
	}
	// if phoneNumberDetails is nil now, means it does not exist, no need to check
	if phoneNumberDetails != nil {
		if phoneNumber.Status != types.WAPhoneNumberStatus(phoneNumberDetails.Status) {
			if err := dependencies.UnitOfWork.WAPhoneNumberRepository().UpdateFields(ctx, phoneNumber.Id, map[string]any{"status": phoneNumberDetails.Status}); err != nil {
				return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		if phoneNumberDetails.Status != "DISCONNECTED" {
			return dto.NewFailedResponse[any](http.StatusBadRequest, "you need to disconnect your phone number first", nil)
		}
	}
	// remove using meta API, but don't delete from db
	// Meta don't allow removing phone number via API
	// https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/phone-numbers#delete-phone-number-from-a-waba
	phoneNumber.Status = types.WAPhoneNumberStatusRemoved
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Remove WhatsApp phone number",
		"Removes an assigned WhatsApp phone number from WhatsApp and the local account. Meta does not allow removing phone number via API. Only MASTER user can access this endpoint.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/wa/phone-numbers/:phone_number_id",
		true,
		true,
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
