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

type PassControl struct {
	CustomerId  int32 `form:"customer_id" val:"required" description:"id of the customer"`
	ToAgent     bool  `form:"to_agent" description:"true to pass control to the AI agent, false to return control to the human"`
	FromWebhook bool  `json:"-" form:"-"`
}

func (passControl *PassControl) Validate() []exception.InputException {
	if passControl.CustomerId <= 0 {
		return []exception.InputException{exception.NewInputException("customer_id", "invalid customer id")}
	}
	return nil
}

func (passControl PassControl) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if !passControl.FromWebhook {
		if user == nil {
			return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
		}
		if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	}
	if inputErrors := passControl.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, passControl.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "customer not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, customer.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var businessAccessToken string
	if !passControl.FromWebhook {
		if user.Type == types.UserTypeOperator {
			if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
				return assignedPhoneNumber.Id == customer.PhoneNumberId
			}) {
				return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
			}
		} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
		businessAccessToken = user.WA.BusinessPortfolioAccessToken
	} else {
		businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetById(ctx, phoneNumber.BusinessAccountId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
			}
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetById(ctx, businessAccount.BussinessPortfolioId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
			}
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		businessAccessToken = businessPortfolio.AccessToken
	}
	if err := dependencies.Facebook.PassControl(ctx, phoneNumber.MetaPhoneNumberId, customer.WAId, businessAccessToken, passControl.ToAgent); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	customer.AgentRunning = passControl.ToAgent
	if err := dependencies.UnitOfWork.CustomerRepository().Update(ctx, customer); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (PassControl) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Pass WhatsApp business agent control",
		"Pass conversation control to the AI agent or return it to the human.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/business-agent/pass-control",
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
