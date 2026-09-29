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

type ListFAQs struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (listFAQs *ListFAQs) Validate() []exception.InputException {
	if listFAQs.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (listFAQs ListFAQs) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]service.AgentFAQ] {
	if user == nil {
		return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := listFAQs.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]service.AgentFAQ](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, listFAQs.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == listFAQs.PhoneNumberId
		}) {
			return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	faqs, err := dependencies.Facebook.ListAgentFAQs(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[[]service.AgentFAQ](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(faqs)
}

func (ListFAQs) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp business agent FAQs",
		"Lists the FAQs configured for a WhatsApp business agent.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/business-agent/faqs",
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
