package feature_customer

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
	Order          types.OrderCustomersType `form:"order"`
	Name           string                   `form:"name"`
	Tags           []string                 `form:"tags"`
	Status         types.CustomerStatus     `form:"status"`
	OnlyHasMessage bool                     `form:"only_has_message"`
	Page           int                      `form:"page"`
	PageSize       int                      `form:"page_size"`
}

func (list *List) Validate() []exception.InputException {
	list.Name = strings.TrimSpace(list.Name)
	if list.Order == "" {
		list.Order = types.OrderCustomersTypeLatestMessage
	}
	for i := range list.Tags {
		list.Tags[i] = strings.TrimSpace(list.Tags[i])
	}
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 10
	}
	inputErrors := []exception.InputException{}
	if list.Order != types.OrderCustomersTypeLatestMessage && list.Order != types.OrderCustomersTypeFromNew && list.Order != types.OrderCustomersTypeFromOld {
		inputErrors = append(inputErrors, exception.NewInputException("order", "invalid order"))
	}
	if list.PageSize > 500 {
		inputErrors = append(inputErrors, exception.NewInputException("page_size", "page size must be less than or equal to 500"))
	}
	if list.Status == "" {
		list.Status = types.CustomerStatusActive
	}
	return inputErrors
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_customer.Customer]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.Customer]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || len(user.WA.PhoneNumbers) == 0 {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.Customer]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_customer.Customer]](inputErrors)
	}
	customers, totalItems, totalPages, err := dependencies.UnitOfWork.CustomerRepository().List(ctx, list.OnlyHasMessage, user.WA.PhoneNumberIds(), list.Name, list.Order, list.Status, list.Tags, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.Customer]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_customer.Customer, 0, len(customers))
	for _, customer := range customers {
		items = append(items, dto_customer.NewCustomer(customer))
	}
	response := dto.NewPagedListResponse(items, totalPages, totalItems)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List customers",
		"Lists customers for the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/customers",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
