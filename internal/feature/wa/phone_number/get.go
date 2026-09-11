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

type Get struct {
	Id int32 `uri:"id" description:"id of the phone number"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "missing phone number ID")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppPhoneNumberDetailsResponse] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppPhoneNumberDetailsResponse](inputErrors)
	}
	if user.WA == nil || user.WA.PhoneNumber_ == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	if user.Type != types.UserTypeMaster {
		get.Id = user.WA.PhoneNumber_.Id
	}
	storedPhoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusNotFound, "phone number not found")
		}
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if storedPhoneNumber.MetaBusinessPortfolioId != user.WA.BusinessPortfolio.MetaBusinessPortfolioId || storedPhoneNumber.MetaWABAId != user.WA.BusinessAccount.MetaWABAId {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	// get phone number from meta
	metaPhoneNumber, err := dependencies.Whatsapp.GetPhoneNumber(ctx, storedPhoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusBadGateway, err.Error())
	}
	// update status in db if needed
	var newStatus types.WAPhoneNumberStatus
	if metaPhoneNumber.Status == "CONNECTED" && storedPhoneNumber.Status != types.WAPhoneNumberStatusConnected {
		newStatus = types.WAPhoneNumberStatusConnected
	} else if metaPhoneNumber.Status != "CONNECTED" && storedPhoneNumber.Status == types.WAPhoneNumberStatusConnected {
		newStatus = types.WAPhoneNumberStatusDisconnected
	}
	if newStatus != "" {
		storedPhoneNumber.Status = newStatus
		// ignore any error
		dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, storedPhoneNumber)
	}
	return dto.NewSuccessResponse(metaPhoneNumber)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp phone number",
		"Gets the current WhatsApp phone number status and configuration from Meta.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/phone-numbers/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
