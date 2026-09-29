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

type UpdateBusinessInfo struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
	*service.AgentPhoneNumberBusinessInfo
}

func (updateBusinessInfo *UpdateBusinessInfo) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if updateBusinessInfo.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	if updateBusinessInfo == nil {
		inputErrors = append(inputErrors, exception.NewInputException("business_info", "missing business info"))
	} else {
		inputErrors = append(inputErrors, updateBusinessInfo.AgentPhoneNumberBusinessInfo.Validate()...)
	}
	return inputErrors
}

func (updateBusinessInfo UpdateBusinessInfo) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := updateBusinessInfo.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, updateBusinessInfo.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == updateBusinessInfo.PhoneNumberId
		}) {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.Facebook.UpdateAgentBusinessInfo(ctx, phoneNumber.MetaPhoneNumberId, updateBusinessInfo.AgentPhoneNumberBusinessInfo, user.WA.BusinessPortfolioAccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (UpdateBusinessInfo) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business agent business info",
		"Updates the business information configured for a WhatsApp business agent.",
		types.HttpRequestTypeQueryJSON,
		http.MethodPut,
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
