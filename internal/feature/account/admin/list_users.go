package feature_account_admin

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

type ListUsers struct {
}

func (ListUsers) Validate() []exception.InputException {
	return nil
}

func (listUsers ListUsers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := listUsers.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_account.User](inputErrors)
	}
	users, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, user.WA.BusinessAccount.Id, nil, true)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var ds []dto_account.User = make([]dto_account.User, len(users))
	for i := range users {
		ds[i] = dto_account.NewUser(users[i])
	}
	return dto.NewSuccessResponse(ds)
}

func (ListUsers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List all users",
		"Lists all users in the business account. Only master users can access this endpoint.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/admin/users",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
