package feature_wa_phone_number

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Delete struct {
	PhoneNumberId int32 `json:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user.Type != types.UserTypeAdmin {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to remove this phone number")
	}
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	phoneNumber, ex := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, delete.PhoneNumberId)
	if ex != nil {
		return dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
	}
	phoneNumbers, ex := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
	}
	authorized := false
	for _, assignedPhoneNumber := range phoneNumbers {
		if assignedPhoneNumber.Id == phoneNumber.Id {
			authorized = true
			break
		}
	}
	if !authorized {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authorized to remove this phone number")
	}
	businessPortfolio, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, phoneNumber.MetaBusinessPortfolioId)
	if ex != nil {
		return dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
	}
	if err := dependencies.Whatsapp.RemovePhoneNumber(ctx, phoneNumber.MetaPhoneNumberId, businessPortfolio.MetaBusinessPortfolioId); err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumber.MetaPhoneNumberId)
		return dto.NewFailedResponse[any](http.StatusBadGateway, "failed to remove phone number from WhatsApp")
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().DeleteById(ctx, phoneNumber.Id); err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumber.Id)
		return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to remove phone number")
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Remove WhatsApp phone number",
		"Removes an assigned WhatsApp phone number from WhatsApp and the local account.",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/wa/v1/phone-numbers",
		true,
		false,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to remove this phone number", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("failed to remove phone number from WhatsApp", http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException("failed to remove phone number", http.StatusInternalServerError)),
		},
	)
}
