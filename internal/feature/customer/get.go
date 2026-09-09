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

type Get struct {
	WAId       string `form:"wa_id" description:"country code + phone number of the customer" example:"6590909090"`
	MetaUserId string `form:"meta_user_id" description:"Meta user id of the customer"`
}

func (get *Get) Validate() []exception.InputException {
	get.WAId = strings.TrimSpace(get.WAId)
	get.MetaUserId = strings.TrimSpace(get.MetaUserId)
	inputErrors := []exception.InputException{}
	if get.WAId == "" && get.MetaUserId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("wa_id", "WA id or Meta user ID is required"))
	}
	return inputErrors
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Customer] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Customer](inputErrors)
	}
	var customer *dto_customer.Customer
	if get.MetaUserId != "" {
		item, err := dependencies.UnitOfWork.CustomerRepository().GetByMetaUserId(ctx, user.Id, get.MetaUserId)
		if err != nil {
			return customerGetError(err)
		}
		mapped := dto_customer.NewCustomer(*item)
		customer = &mapped
	} else {
		item, err := dependencies.UnitOfWork.CustomerRepository().GetByWAId(ctx, user.Id, get.WAId)
		if err != nil {
			return customerGetError(err)
		}
		mapped := dto_customer.NewCustomer(*item)
		customer = &mapped
	}
	return dto.NewSuccessResponse(customer)
}

func customerGetError(err error) dto.Response[*dto_customer.Customer] {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusNotFound, "customer not found")
	}
	return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get customer",
		"Gets a customer by WA ID or Meta user ID.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/customers/get",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("customer not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
