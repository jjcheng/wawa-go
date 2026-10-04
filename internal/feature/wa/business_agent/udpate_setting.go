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

type UpdateSetting struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
	service.AgentSetting
}

func (updateSetting *UpdateSetting) Validate() []exception.InputException {
	var errors []exception.InputException
	if updateSetting.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	errors = append(errors, updateSetting.AgentSetting.Validate()...)
	return errors
}

func (updateSetting UpdateSetting) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentSetting] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentSetting](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentSetting](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := updateSetting.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentSetting](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, updateSetting.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentSetting](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentSetting](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == updateSetting.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentSetting](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentSetting](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// don't change it's turned on/off, use switches to turn on/off
	updateSetting.Rollout.Enabled = phoneNumber.AgentEnabled
	updatedSetting, err := dependencies.Facebook.UpdateSetting(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, &updateSetting.AgentSetting)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentSetting](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(updatedSetting)
}

func (UpdateSetting) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business agent settings",
		"Updates the configuration settings for a WhatsApp business agent.",
		types.HttpRequestTypeQueryJSON,
		http.MethodPut,
		"/v1/wa/business-agent/settings",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
