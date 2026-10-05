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

type GetWebsite struct {
	PhoneNumberId int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	Id            string `uri:"id" val:"required" description:"id of the website"`
}

func (getWebsite *GetWebsite) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if getWebsite.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	getWebsite.Id = strings.TrimSpace(getWebsite.Id)
	if getWebsite.Id == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing website ID"))
	}
	return inputErrors
}

func (getWebsite GetWebsite) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentWebsite] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentWebsite](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentWebsite](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getWebsite.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentWebsite](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, getWebsite.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentWebsite](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentWebsite](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == getWebsite.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentWebsite](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentWebsite](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	website, err := dependencies.Facebook.GetAgentWebsite(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, getWebsite.Id)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentWebsite](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(website)
}

func (GetWebsite) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business agent website",
		"Gets a website configured for a WhatsApp business agent.",
		types.HttpRequestTypeUri,
		http.MethodGet,
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
