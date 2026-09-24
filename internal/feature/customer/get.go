package feature_customer

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	Id int32 `uri:"id" val:"required" description:"id of the customer"`
}

func (get *Get) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if get.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing id"))
	}
	return inputErrors
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Customer] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Customer](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusNotFound, "customer not found", nil)
		}
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == customer.PhoneNumberId
	}) {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	d := dto_customer.NewCustomer(*customer)
	return dto.NewSuccessResponse(&d)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get customer",
		"Gets a customer by ID.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/customers/:id",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("customer not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
