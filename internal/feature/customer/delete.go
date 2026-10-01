package feature_customer

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Delete struct {
	Ids []int32 `json:"ids" val:"required" description:"id of the customers to delete"`
}

func (delete *Delete) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if len(delete.Ids) == 0 {
		return append(inputErrors, exception.NewInputException("ids", "missing customer ids"))
	}
	seen := map[int32]struct{}{}
	for _, id := range delete.Ids {
		if id <= 0 {
			inputErrors = append(inputErrors, exception.NewInputException("ids", "customer ids must be positive"))
			continue
		}
		if _, exists := seen[id]; exists {
			inputErrors = append(inputErrors, exception.NewInputException("ids", "duplicate customer id"))
			continue
		}
		seen[id] = struct{}{}
	}
	return inputErrors
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := delete.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	customers, err := transaction.CustomerRepository().ListByPhoneNumberIdsAndIds(ctx, user.WA.PhoneNumberIds(), delete.Ids)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(customers) != len(delete.Ids) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err = transaction.CustomerRepository().DeleteByIds(ctx, delete.Ids); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete a customers",
		"Delete a customer, only MASTER user can access this endpoint.",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/v1/customers",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to delete one or more customers", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
