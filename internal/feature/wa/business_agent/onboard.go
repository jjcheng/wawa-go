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

type Onboard struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

type OnboardResult struct {
	AgentId string `json:"agent_id"`
}

func (onboard *Onboard) Validate() []exception.InputException {
	if onboard.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (onboard Onboard) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*OnboardResult] {
	if user == nil {
		return dto.NewFailedResponse[*OnboardResult](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*OnboardResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := onboard.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*OnboardResult](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, onboard.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*OnboardResult](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*OnboardResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// for operator need to check if the number is managed, otherwise check business account
	if user.Type != types.UserTypeMaster && !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == onboard.PhoneNumberId
	}) {
		return dto.NewFailedResponse[*OnboardResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*OnboardResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	agentId, err := dependencies.Facebook.OnboardAgent(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*OnboardResult](http.StatusBadGateway, err.Error(), err)
	}
	if strings.TrimSpace(agentId) == "" {
		return dto.NewFailedResponse[*OnboardResult](http.StatusBadGateway, "agent ID is missing from Meta response", nil)
	}
	phoneNumber.MetaAgentId = agentId
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		return dto.NewFailedResponse[*OnboardResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if dependencies.AuthCache != nil {
		dependencies.AuthCache.InvalidateUser(user.Id)
	}
	return dto.NewSuccessResponse(&OnboardResult{AgentId: agentId})
}

func (Onboard) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Onboard WhatsApp business agent",
		"Onboards a WhatsApp phone number with the business agent and stores the returned Meta agent ID.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/business-agent/onboard",
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
