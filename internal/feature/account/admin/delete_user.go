package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type DeleteUser struct {
	Id int32 `uri:"id" val:"required" description:"id of the user"`
}

func (deleteUser *DeleteUser) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if deleteUser.Id <= 0 {
		errors = append(errors, exception.NewInputException("id", "missing id"))
	}
	return errors
}

func (deleteUser DeleteUser) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := deleteUser.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	existingUser, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, deleteUser.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if existingUser.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if existingUser.Type == types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "master user cannot be deleted", nil)
	}
	err = dependencies.UnitOfWork.AccountUserRepository().DeleteById(ctx, existingUser.Id)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (DeleteUser) APISettings() feature.APISettings {
	return feature.NewAPISettings("Delete a user", "Delete a user by id. Only MASTER can delete non-MASTER users.", types.HttpRequestTypeUri, "DELETE", "/v1/admin/users/:id", true, true, types.APITagAdmin, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("master user cannot be deleted", http.StatusBadRequest)),
	}, feature.NewAIWorker(true,
		"To delete a user, go to Assets -> Users and select the user to delete. Only MASTER users can do this, and MASTER users cannot be deleted.",
		types.AIWorkerReturnTypeText,
		"User successfully deleted.",
		"/assets/users",
		feature.NewAIWorkerRequire("Select a user to delete",
			ListUsers{}, dto_ai_worker.WorkInput{
				Name:               "id",
				Description:        "user to delete",
				Type:               types.AIInputFieldTypeInt,
				ReferenceFieldName: "id",
				DisplayType:        types.AIWorkerDisplayTypeSingleChoiceTable,
			}),
	))
}
