package feature_account_admin

import (
	"context"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
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
	users, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, user.BusinessAccountId, nil)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var ds []dto_account.User = make([]dto_account.User, len(users))
	for i := range users {
		ds[i] = dto_account.NewUser(users[i])
	}
	// get assigned phone nubers
	if len(ds) > 0 {
		userIds := helper.Map(ds, func(u dto_account.User) int32 {
			return u.Id
		})
		userPhoneNumbers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByUserIds(ctx, userIds)
		if err != nil {
			return dto.NewFailedResponse[[]dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		for i := range ds {
			items := helper.Filter(userPhoneNumbers, func(up dao_account.UserPhoneNumber) bool {
				return up.UserId == ds[i].Id
			})
			ds[i].AssignedPhoneNumbers = helper.Map(items, func(up dao_account.UserPhoneNumber) dto_account.AssignedPhoneNumber {
				return dto_account.AssignedPhoneNumber{
					Id:                 up.PhoneNumberId,
					Name:               up.PhoneNumber.Name,
					DisplayPhoneNumber: up.PhoneNumber.DisplayPhoneNumber,
					MetaPhoneNumberId:  up.PhoneNumber.MetaPhoneNumberId,
					AgentRunning:       up.PhoneNumber.AgentRunning,
					MetaAgentId:        up.PhoneNumber.MetaAgentId,
				}
			})
		}
	}
	return dto.NewSuccessResponse(ds)
}

func (ListUsers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List all users",
		"List all users in the business account. Only master users can access this endpoint.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/admin/users",
		true,
		true,
		types.APITagAdmin,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
		feature.NewAIWorker(true, "To view all users in your business account, go to Assets / Users.", types.AIWorkerReturnTypeData, "", "/assets/users"),
	)
}
