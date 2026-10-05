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

type Test struct {
	PhoneNumberId  int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	UserMessage    string `json:"user_message" val:"required" description:"user's latest message"`
	ConversationId string `json:"conversation_id" description:"id of the conversation, empty if it's a new conversation"`
}

func (test *Test) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if test.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	test.UserMessage = strings.TrimSpace(test.UserMessage)
	if test.UserMessage == "" {
		inputErrors = append(inputErrors, exception.NewInputException("user_message", "missing user message"))
	}
	test.ConversationId = strings.TrimSpace(test.ConversationId)
	return inputErrors
}

func (test Test) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentTestResponse] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := test.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentTestResponse](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, test.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == test.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result, err := dependencies.Facebook.TestAgent(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, test.UserMessage, test.ConversationId)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentTestResponse](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(result)
}

func (Test) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Test WhatsApp business agent",
		"Sends a test user message to a WhatsApp business agent and returns its response.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/test",
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
