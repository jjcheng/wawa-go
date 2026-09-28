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

type GetBusinessProfile struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (getBusinessProfile *GetBusinessProfile) Validate() []exception.InputException {
	return nil
}

func (getBusinessProfile GetBusinessProfile) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppPhoneNumberBusinessProfileData] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := getBusinessProfile.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppPhoneNumberBusinessProfileData](errors)
	}
	if getBusinessProfile.PhoneNumberId <= 0 {
		getBusinessProfile.PhoneNumberId = user.WA.PhoneNumbers[0].Id
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getBusinessProfile.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if phoneNumber == nil || phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	businessProfile, err := dependencies.Whatsapp.GetPhoneNumberBusinessProfile(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusBadGateway, err.Error(), err)
	}
	if len(businessProfile.Data) == 0 {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusBadGateway, "no business profile", nil)
	}
	return dto.NewSuccessResponse(&businessProfile.Data[0])
}

func (GetBusinessProfile) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get business profile",
		"Get business profile using WhatsApp phone number.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/phone-numbers/business-profile",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("no business profile", http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
