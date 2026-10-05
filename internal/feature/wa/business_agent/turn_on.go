package feature_wa_business_agent

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

type TurnOn struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"phone number id to turn on/off agent"`
	On            bool  `form:"on" description:"true to enable, false to disable"`
}

func (turnOn *TurnOn) Validate() []exception.InputException {
	if turnOn.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (turnOn TurnOn) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == turnOn.PhoneNumberId
	}) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := turnOn.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, turnOn.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	err = dependencies.Facebook.TurnAgentOnOff(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, turnOn.On)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if phoneNumber.AgentEnabled != turnOn.On {
		phoneNumber.AgentEnabled = turnOn.On
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if dependencies.AuthCache != nil {
			dependencies.AuthCache.InvalidateUser(user.Id)
		}
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (TurnOn) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Turn WhatsApp business agent on/off",
		"Enable or disable the business agent for a WhatsApp phone number.",
		types.HttpRequestTypeUriQuery,
		http.MethodPatch,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/status",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
