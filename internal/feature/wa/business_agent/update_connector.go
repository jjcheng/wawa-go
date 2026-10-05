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

type UpdateConnector struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	service.AgentConnector
}

func (updateConnector *UpdateConnector) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	updateConnector.ID = strings.TrimSpace(updateConnector.ID)
	if updateConnector.ID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing connector ID"))
	}
	if updateConnector.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	return inputErrors
}

func (updateConnector UpdateConnector) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentConnector] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := updateConnector.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentConnector](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, updateConnector.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentConnector](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == updateConnector.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	connector, err := dependencies.Facebook.UpdateAgentConnector(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, &updateConnector.AgentConnector)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(connector)
}

func (UpdateConnector) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business agent connector",
		"Updates a connector configured for a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/connectors",
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
