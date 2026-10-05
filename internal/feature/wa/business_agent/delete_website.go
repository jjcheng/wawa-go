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

type DeleteWebsite struct {
	PhoneNumberId int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	Id            string `uri:"id" val:"required" description:"id of the website"`
}

func (deleteWebsite *DeleteWebsite) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if deleteWebsite.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	deleteWebsite.Id = strings.TrimSpace(deleteWebsite.Id)
	if deleteWebsite.Id == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing website ID"))
	}
	return inputErrors
}

func (deleteWebsite DeleteWebsite) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := deleteWebsite.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, deleteWebsite.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == deleteWebsite.PhoneNumberId
		}) {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.Facebook.DeleteAgentWebsite(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, deleteWebsite.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (DeleteWebsite) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp business agent website",
		"Deletes a website configured for a WhatsApp business agent.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/websites/:id",
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
