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

type CreateConnector struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	service.AgentConnector
}

func (createConnector *CreateConnector) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if createConnector.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid phone number id"))
	}
	createConnector.Name = strings.TrimSpace(createConnector.Name)
	if createConnector.Name == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "missing connector name"))
	}
	createConnector.BaseURL = strings.TrimSpace(createConnector.BaseURL)
	if createConnector.BaseURL == "" {
		inputErrors = append(inputErrors, exception.NewInputException("base_url", "missing connector base URL"))
	}
	createConnector.ConnectorProtocol = strings.TrimSpace(createConnector.ConnectorProtocol)
	if createConnector.ConnectorProtocol == "" {
		inputErrors = append(inputErrors, exception.NewInputException("connector_protocol", "missing connector protocol"))
	}
	createConnector.AuthType = strings.TrimSpace(createConnector.AuthType)
	if createConnector.AuthType == "" {
		inputErrors = append(inputErrors, exception.NewInputException("auth_type", "missing connector auth type"))
	}
	return inputErrors
}

func (createConnector CreateConnector) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentConnector] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := createConnector.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentConnector](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, createConnector.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentConnector](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == createConnector.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	connector, err := dependencies.Facebook.CreateAgentConnector(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, &createConnector.AgentConnector)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentConnector](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(connector)
}

func (CreateConnector) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp business agent connector",
		"Creates a connector for a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
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
