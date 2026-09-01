package feature_wa_user_phone_number

import (
	"context"
	"errors"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	UserId        int32 `json:"user_id" val:"required" description:"if of the user who will manage this phone number"`
	PhoneNumberId int32 `json:"phone_number_id" val:"required" description:"phone number if this user will manage"`
}

func (create *Create) Validate() []exception.InputException {
	var errors []exception.InputException
	if create.UserId <= 0 {
		errors = append(errors, exception.NewInputException("user_id", "missing user id"))
	}
	if create.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "missing phone number id"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.UserPhoneNumber] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.UserPhoneNumber](errors)
	}
	if _, err := dependencies.UnitOfWork.AccountUserRepository().Get(ctx, create.UserId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusNotFound, "user not found")
		}
		return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, create.PhoneNumberId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusNotFound, "phone number not found")
		}
		return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// check exist
	existing, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetByUserIdPhoneNumberId(ctx, create.UserId, create.PhoneNumberId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	if existing != nil {
		d := dto_wa.NewUserPhoneNumber(*existing)
		return dto.NewSuccessResponse(&d)
	}
	userPhoneNumber := dao_wa.UserPhoneNumber{
		UserId:        create.UserId,
		PhoneNumberId: create.PhoneNumberId,
	}
	if err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().Insert(ctx, &userPhoneNumber); err != nil {
		return dto.NewFailedResponse[*dto_wa.UserPhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	d := dto_wa.NewUserPhoneNumber(userPhoneNumber)
	return dto.NewSuccessResponse(&d)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create user phone number assignment",
		"Assigns a WhatsApp phone number to a user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/user-phone-numbers",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
