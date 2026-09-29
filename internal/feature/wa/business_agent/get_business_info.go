package feature_wa_business_agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

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

type GetBusinessInfo struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (getBusinessInfo *GetBusinessInfo) Validate() []exception.InputException {
	if getBusinessInfo.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (getBusinessInfo GetBusinessInfo) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentPhoneNumberBusinessInfo] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getBusinessInfo.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentPhoneNumberBusinessInfo](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getBusinessInfo.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == getBusinessInfo.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	businessInfo, err := dependencies.Facebook.GetAgentBusinessInfo(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentPhoneNumberBusinessInfo](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(businessInfo)
}

func (GetBusinessInfo) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business agent business info",
		"Gets the business information configured for a WhatsApp business agent.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/business-agent/business-info",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
