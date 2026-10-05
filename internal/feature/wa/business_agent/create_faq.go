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

type CreateFAQ struct {
	PhoneNumberId int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	Question      string `json:"question" val:"required" description:"question of the FAQ"`
	Answer        string `json:"answer" val:"required" description:"answer of the FAQ"`
}

func (createFAQ *CreateFAQ) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if createFAQ.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	createFAQ.Question = strings.TrimSpace(createFAQ.Question)
	if createFAQ.Question == "" {
		inputErrors = append(inputErrors, exception.NewInputException("question", "missing FAQ question"))
	}
	createFAQ.Answer = strings.TrimSpace(createFAQ.Answer)
	if createFAQ.Answer == "" {
		inputErrors = append(inputErrors, exception.NewInputException("answer", "missing FAQ answer"))
	}
	return inputErrors
}

func (createFAQ CreateFAQ) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentFAQ] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentFAQ](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := createFAQ.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentFAQ](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, createFAQ.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentFAQ](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentFAQ](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == createFAQ.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	faq, err := dependencies.Facebook.CreateAgentFAQ(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, &service.AgentFAQ{
		Question: createFAQ.Question,
		Answer:   createFAQ.Answer,
	})
	if err != nil {
		return dto.NewFailedResponse[*service.AgentFAQ](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(faq)
}

func (CreateFAQ) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp business agent FAQ",
		"Creates an FAQ for a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/faqs",
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
