package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// for master user to update user profile
type GetUser struct {
	Id int32 `uri:"id" val:"required" description:"id of the user"`
}

func (getUser *GetUser) Validate() []exception.InputException {
	if getUser.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid user id")}
	}
	return nil
}

func (getUser GetUser) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getUser.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](inputErrors)
	}
	targetUser, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, getUser.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "user not found", nil)
		}
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if targetUser.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "user not found", nil)
	}
	// get assigned phone numbers
	userPhoneNumbers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByUserId(ctx, getUser.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_account.NewUser(*targetUser)
	result.AssignedPhoneNumbers = helper.Map(userPhoneNumbers, func(up dao_account.UserPhoneNumber) dto_account.AssignedPhoneNumber {
		return dto_account.AssignedPhoneNumber{
			Id:                 up.PhoneNumberId,
			Name:               up.PhoneNumber.Name,
			DisplayPhoneNumber: up.PhoneNumber.DisplayPhoneNumber,
			MetaPhoneNumberId:  up.PhoneNumber.MetaPhoneNumberId,
			AgentEnabled:       up.PhoneNumber.AgentEnabled,
			MetaAgentId:        up.PhoneNumber.MetaAgentId,
		}
	})
	return dto.NewSuccessResponse(&result)
}

func (GetUser) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a user",
		"Gets a user by id. Only master users can access this endpoint",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/admin/users/:id",
		true,
		true,
		types.APITagAdmin,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
		feature.NewAIWorker(true,
			"To view a user's details, go to Assets -> Users and select the user.",
			types.AIWorkerReturnTypeData,
			"",
			"/assets/users",
			feature.NewAIWorkerRequire("Select a user", ListUsers{}, dto_ai_worker.WorkInput{
				Name:               "id",
				Description:        "user to view",
				Type:               types.AIInputFieldTypeInt,
				ReferenceFieldName: "id",
				DisplayType:        types.AIWorkerDisplayTypeSingleChoiceTable,
			}),
		),
	)
}
