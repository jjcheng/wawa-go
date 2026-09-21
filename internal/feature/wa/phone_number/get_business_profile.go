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

type GetBusinessProfile struct {
}

func (getBusinessProfile *GetBusinessProfile) Validate() []exception.InputException {
	return nil
}

func (getBusinessProfile GetBusinessProfile) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppPhoneNumberBusinessProfileData] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusForbidden, "you are not authenticated")
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil || user.WA.PhoneNumber_ == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusUnauthorized, "you are not authorized")
	}
	businessProfile, err := dependencies.Whatsapp.GetPhoneNumberBusinessProfile(ctx, user.WA.PhoneNumber_.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err)
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusBadGateway, err.Error())
	}
	if len(businessProfile.Data) == 0 {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberBusinessProfileData](http.StatusBadGateway, "no business profile")
	}
	return dto.NewSuccessResponse(&businessProfile.Data[0])
}

func (GetBusinessProfile) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get business profile",
		"Get business profile using WhatsApp phone number.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/wa/phone-numbers/business-profile",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("no business profile", http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
