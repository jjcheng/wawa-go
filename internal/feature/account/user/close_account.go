package feature_account_user

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CloseAccount struct {
	Reason      string                         `json:"reason" val:"required" description:"reason for closing account"`
	ReasonTypes []types.CloseAccountReasonType `json:"reason_types" description:"select 1 or more preset reasons"`
}

func (closeAccount *CloseAccount) Validate() []exception.InputException {
	closeAccount.Reason = strings.TrimSpace(closeAccount.Reason)
	inputErrors := []exception.InputException{}
	return inputErrors
}

func (closeAccount CloseAccount) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := closeAccount.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	// get all MASTER users in the business account
	if user.Type == types.UserTypeMaster {
		users, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, user.BusinessAccountId, nil)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		// if there is more than 1 user, and current user is master, and business account only has 1 master, ask user to assign another master first
		if len(users) > 1 {
			numberOfMasters := helper.Count(users, func(u dao_account.User) bool {
				return u.Type == types.UserTypeMaster
			})
			if numberOfMasters == 1 { // has to be the current user
				return dto.NewFailedResponse[any](http.StatusBadRequest, "You are the only MASTER user in your business account, please assign a new MASTER user before closing your account in Settings / Users.", nil)
			}
		}
	}
	// begin a transaction
	var committed bool
	transaction := dependencies.UnitOfWork.BeginTransaction()
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	// set user to closed
	userDAO, err := transaction.AccountUserRepository().GetById(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	userDAO.CloseReason = closeAccount.Reason
	userDAO.CloseReasonTypes = make([]string, len(closeAccount.ReasonTypes))
	for i, reasonType := range closeAccount.ReasonTypes {
		userDAO.CloseReasonTypes[i] = string(reasonType)
	}
	userDAO.Status = types.UserStatusClosed
	userDAO.PasswordHash = uuid.NewString()
	if err := transaction.AccountUserRepository().Update(ctx, userDAO); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// delete login sessions, no need to return error
	if err := transaction.AccountSessionRepository().DeleteByUserId(ctx, user.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	// invalidate the auth_cache
	if dependencies.AuthCache != nil {
		dependencies.AuthCache.InvalidateUser(user.Id)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

// don't use this, will delete everything
// func (closeAccount *CloseAccount) _(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
// 	// start a transaction
// 	transaction := dependencies.UnitOfWork.BeginTransaction()
// 	var committed bool
// 	defer func() {
// 		if !committed {
// 			transaction.Rollback()
// 		}
// 	}()
// 	// broadcasts, will also delete broadcast_recipients using FK
// 	if err := transaction.BroadcastRepository().DeleteByUserId(ctx, user.Id); err != nil {
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 	}
// 	// messages, will delete message_status using FK
// 	if user.WA != nil && user.WA.PhoneNumber_ != nil {
// 		if err := transaction.WAMessageRepository().DeleteByPhoneNumberId(ctx, user.WA.PhoneNumber_.Id); err != nil {
// 			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 		}
// 		// delete phone number
// 		if err := transaction.WAPhoneNumberRepository().DeleteById(ctx, user.WA.PhoneNumber_.Id); err != nil {
// 			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 		}
// 	}
// 	// customers
// 	if err := transaction.CustomerRepository().DeleteByUserId(ctx, user.Id); err != nil {
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 	}
// 	// user
// 	if err := transaction.AccountUserRepository().DeleteById(ctx, user.Id); err != nil {
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 	}
// 	// check how many users left in business account
// 	if user.WA != nil && user.WA.BusinessAccount != nil {
// 		users, err := transaction.AccountUserRepository().ListByBusinessAccountId(ctx, user.WA.BusinessAccount.Id, nil, true)
// 		if err != nil {
// 			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 		}
// 		if len(users) == 0 {
// 			// if no more user, delete business account
// 			if err := transaction.WABusinessAccountRepository().DeleteById(ctx, user.WA.BusinessAccount.Id); err != nil {
// 				return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 			}
// 			// if only business portfolio has no business account, delete it
// 			if user.WA.BusinessPortfolio != nil {
// 				businessAccounts, err := transaction.WABusinessAccountRepository().ListByBusinessPortfolioId(ctx, user.WA.BusinessPortfolio.Id)
// 				if err != nil {
// 					return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 				}
// 				if len(businessAccounts) == 0 {
// 					if err := transaction.WABusinessPortfolioRepository().DeleteById(ctx, user.WA.BusinessPortfolio.Id); err != nil {
// 						return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 					}
// 				}
// 			}

// 		}
// 	}
// 	// commit
// 	if err := transaction.CommitTransaction(); err != nil {
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
// 	}
// 	committed = true
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

func (CloseAccount) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Close account",
		"Delete everything of the user from DB",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/v1/account/users/me",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
