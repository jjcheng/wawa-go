package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type SetUserType struct {
	UserId int            `json:"id" val:"required" description:"id of the user"`
	Type   types.UserType `json:"type" val:"required" description:"new type of the user"`
}

func (setUserType *SetUserType) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if setUserType.UserId <= 0 {
		errors = append(errors, exception.NewInputException("id", "missing user id"))
	}
	if setUserType.Type != types.UserTypeMaster && setUserType.Type != types.UserTypeOperator {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	return errors
}

func (setUserType SetUserType) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not master")
	}
	if user.WA == nil {
		return dto.NewFailedResponse[any](http.StatusNotFound, "user's WhatsApp not found")
	}
	if errors := setUserType.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	if user.Id == int32(setUserType.UserId) {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "you cannot update status of yourself")
	}
	existingUser, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, int32(setUserType.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "user not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if existingUser.Type == types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized")
	}
	targetBusinessPortfolio, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, int32(setUserType.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "Meta business portfolio not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if user.WA.BusinessPortfolio.MetaBusinessPortfolioId != targetBusinessPortfolio.MetaBusinessPortfolioId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to update this user")
	}
	existingUser.Type = setUserType.Type
	err = dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existingUser)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (SetUserType) APISettings() feature.APISettings {
	return feature.NewAPISettings("Admin update user type", "Set user to MASTER or OPERATOR. Only admin can set user type.", types.HttpRequestTypeJSON, "PATCH", "/v1/admin/user-type", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("you cannot update status of yourself", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException("you are not authorized to update this user", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("Meta business portfolio not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
