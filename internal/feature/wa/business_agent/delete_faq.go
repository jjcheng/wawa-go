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

type DeleteFAQ struct {
	PhoneNumberId int32  `form:"phone_number_id" val:"required" description:"id of the phone number"`
	Id            string `uri:"id" val:"required" description:"id of the faq"`
}

func (deleteFAQ *DeleteFAQ) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if deleteFAQ.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	deleteFAQ.Id = strings.TrimSpace(deleteFAQ.Id)
	if deleteFAQ.Id == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing FAQ ID"))
	}
	return inputErrors
}

func (deleteFAQ DeleteFAQ) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := deleteFAQ.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, deleteFAQ.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == deleteFAQ.PhoneNumberId
		}) {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.Facebook.DeleteAgentFAQ(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, deleteFAQ.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (DeleteFAQ) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp business agent FAQ",
		"Deletes a FAQ configured for a WhatsApp business agent.",
		types.HttpRequestTypeUriQuery,
		http.MethodDelete,
		"/v1/wa/business-agent/faqs/:id",
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
