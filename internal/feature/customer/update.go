package feature_customer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Update struct {
	CustomerId     int32                `uri:"customer_id" val:"required" description:"id of the customer in our db"`
	DisplayName    string               `json:"display_name" val:"required" description:"display name of the customer"`
	CountryCode    string               `json:"country_code" description:"country code if in db it's ."`
	PhoneNumber    string               `json:"phone_number" description:"if country code is . in db"`
	Tags           []string             `json:"tags" description:"tags of the customer"`
	Status         types.CustomerStatus `json:"status" val:"required" description:"status of the customer"`
	Remarks        string               `json:"remarks" description:"for your own reference"`
	AdditionalData map[string]any       `json:"additional_data" description:"other profile data like bd"`
}

func (update *Update) Validate() []exception.InputException {
	update.DisplayName = strings.TrimSpace(update.DisplayName)
	update.CountryCode = strings.TrimSpace(update.CountryCode)
	update.PhoneNumber = strings.TrimSpace(update.PhoneNumber)
	update.Remarks = strings.TrimSpace(update.Remarks)
	if update.Tags == nil {
		update.Tags = []string{}
	} else {
		for index := range update.Tags {
			update.Tags[index] = strings.TrimSpace(update.Tags[index])
		}
	}
	inputErrors := []exception.InputException{}
	if update.CustomerId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("customer_id", "missing customer id"))
	}
	if update.DisplayName == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "missing customer name"))
	}
	return inputErrors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Customer] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Customer](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, update.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusNotFound, "customer not found", nil)
		}
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if customer.UserId != user.Id {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if update.CountryCode != "" {
		if customer.CountryCode != update.CountryCode || customer.PhoneNumber != update.PhoneNumber {
			existing, err := dependencies.UnitOfWork.CustomerRepository().GetByCountryCodePhoneNumber(ctx, user.Id, update.CountryCode, update.PhoneNumber)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			if existing != nil && existing.Id != customer.Id {
				return dto.NewFailedResponse[*dto_customer.Customer](http.StatusConflict, "customer already exists", nil)
			}
			customer.CountryCode = update.CountryCode
			customer.PhoneNumber = update.PhoneNumber
		}
	}
	customer.DisplayName = update.DisplayName
	customer.Tags = update.Tags
	customer.Status = update.Status
	customer.Remarks = update.Remarks
	customer.WAId = customer.CountryCode + customer.PhoneNumber
	customer.AdditionalData = update.AdditionalData
	if err := dependencies.UnitOfWork.CustomerRepository().Update(ctx, customer); err != nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_customer.NewCustomer(*customer)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update customer",
		"Updates a customer for the authenticated user.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/customers/:customer_id",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("customer not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("customer already exists", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
