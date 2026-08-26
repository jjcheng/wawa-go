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

type Create struct {
	Name            string           `json:"name" val:"required" description:"name of the new user" example:"John Doe"`
	PhoneNumber     string           `json:"phone_number" val:"required" description:"phone number of the user"`
	Type            types.UserType   `json:"type" val:"required" description:"type of the user" example:"PUBLIC"`
	Description     string           `json:"description" description:"for your reference" example:"created by account department"`
	Password        string           `json:"password" val:"required" description:"password of the user"`
	ConfirmPassword string           `json:"confirm_password" val:"required" description:"confirm password of the user"`
	Status          types.UserStatus `json:"status" val:"required" description:"status of the user"`
}

func (create *Create) Validate() []exception.InputException {
	create.Name = strings.TrimSpace(create.Name)
	create.Description = strings.TrimSpace(create.Description)
	create.PhoneNumber = strings.TrimSpace(create.PhoneNumber)
	create.Password = strings.TrimSpace(create.Password)
	create.ConfirmPassword = strings.TrimSpace(create.ConfirmPassword)
	errors := []exception.InputException{}
	if create.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if create.PhoneNumber == "" {
		errors = append(errors, exception.NewInputException("phone_number", "missing phone number"))
	}
	if create.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if !helper.Any(types.UserTypes, func(t types.UserType) bool { return t == create.Type }) {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	if create.Status == "" {
		errors = append(errors, exception.NewInputException("status", "missing status"))
	} else if create.Status != types.UserStatusActive && create.Status != types.UserStatusPendingPassword && create.Status != types.UserStatusInactive {
		errors = append(errors, exception.NewInputException("status", "invalid status"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user != nil && user.Type != types.UserTypeAdmin {
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadRequest, "you are not admin")
	}
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// check phone number exists
	existing, ex := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, create.PhoneNumber)
	if ex != nil && ex.StatusCode != 404 {
		return dto.NewFailedResponse[*dto_account.User](ex.StatusCode, ex.Message)
	}
	if existing != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadRequest, "phone number already exists")
	}
	// generate password hash
	passwordHash, err := helper.HashPassword(create.Password)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to generate password hash")
	}
	// create app
	newUser := dao_account.User{
		Name:         create.Name,
		Email:        strings.ReplaceAll(uuid.NewString(), "-", "") + "@coreconcept.tech",
		PhoneNumber:  create.PhoneNumber,
		Description:  create.Description,
		Type:         create.Type,
		PasswordHash: passwordHash,
		Status:       create.Status,
	}
	err = dependencies.UnitOfWork.AccountUserRepository().Insert(ctx, &newUser)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, user.Id, create)
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to create user")
	}
	d := dto_account.NewUser(newUser)
	return dto.NewSuccessResponse(&d)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings("Create new user", "Allow user to login using password", types.HttpRequestTypeJSON, "POST", "/account/users/v1", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not admin", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("failed to generate password hash", http.StatusInternalServerError)),
		feature.NewAPIError(*exception.NewCustomException("failed to create user", http.StatusInternalServerError)),
	})
}
