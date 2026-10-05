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

type GetConnectorLog struct {
	PhoneNumberId int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	ConnectorId   string `uri:"connector_id" val:"required" description:"id of the connector"`
}

func (getConnectorLog *GetConnectorLog) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if getConnectorLog.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid phone number id"))
	}
	getConnectorLog.ConnectorId = strings.TrimSpace(getConnectorLog.ConnectorId)
	if getConnectorLog.ConnectorId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("connector_id", "missing connector ID"))
	}
	return inputErrors
}

func (getConnectorLog GetConnectorLog) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentConnectorLog] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getConnectorLog.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentConnectorLog](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getConnectorLog.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == getConnectorLog.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	connectorLog, err := dependencies.Facebook.ListAgentConnectorLogs(ctx, phoneNumber.MetaPhoneNumberId, getConnectorLog.ConnectorId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentConnectorLog](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(connectorLog)
}

func (GetConnectorLog) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business agent connector logs",
		"Gets logs and statistics for a WhatsApp business agent connector.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/connectors/:connector_id/logs",
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
