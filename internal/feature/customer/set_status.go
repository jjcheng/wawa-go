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

type SetStatus struct {
	Ids    []int32              `json:"ids" val:"required" description:"id of the customers"`
	Status types.CustomerStatus `json:"status" val:"required" description:"status of the customers"`
}

func (setStatus *SetStatus) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if len(setStatus.Ids) == 0 {
		return append(inputErrors, exception.NewInputException("ids", "missing customer ids"))
	}
	seen := map[int32]struct{}{}
	for _, id := range setStatus.Ids {
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
	if setStatus.Status != types.CustomerStatusActive && setStatus.Status != types.CustomerStatusInactive {
		inputErrors = append(inputErrors, exception.NewInputException("status", "invalid customer status"))
	}
	return inputErrors
}

func (setStatus SetStatus) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := setStatus.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	customerCount, err := transaction.CustomerRepository().CountByIds(ctx, user.Id, setStatus.Ids)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if customerCount != len(setStatus.Ids) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to update one or more customers")
	}
	for _, id := range setStatus.Ids {
		if err := transaction.CustomerRepository().UpdateFields(ctx, id, map[string]any{"status": setStatus.Status}); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (SetStatus) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Set customer status",
		"Updates the status of customers owned by the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPatch,
		"/v1/customers/status",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to update one or more customers", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
