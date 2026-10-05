package feature_wa_phone_number

import (
	"context"
	"errors"
	"net/http"

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

type GetLocal struct {
	Id int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (getLocal *GetLocal) Validate() []exception.InputException {
	if getLocal.Id <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (getLocal GetLocal) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.PhoneNumber] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getLocal.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PhoneNumber](inputErrors)
	}
	if user.Type != types.UserTypeMaster && !helper.Any(user.WA.PhoneNumbers, func(phoneNumber dto_wa.PhoneNumber) bool {
		return phoneNumber.Id == getLocal.Id
	}) {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getLocal.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_wa.NewPhoneNumber(*phoneNumber)
	return dto.NewSuccessResponse(&result)
}

func (GetLocal) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get local WhatsApp phone number",
		"Get a WhatsApp phone number from the local database without calling WhatsApp API.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/phone-numbers/:phone_number_id/local",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
