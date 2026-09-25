package feature_commerce_website

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

type SetStatus struct {
	Id     int32                       `uri:"id" val:"required" description:"id of the website"`
	Status types.CommerceWebsiteStatus `form:"status" val:"required" description:"status to be updated"`
}

func (setStatus *SetStatus) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if setStatus.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid website id"))
	}
	if setStatus.Status != types.CommerceWebsiteStatusActive && setStatus.Status != types.CommerceWebsiteStatusInactive {
		inputErrors = append(inputErrors, exception.NewInputException("status", "invalid website status"))
	}
	return inputErrors
}

func (setStatus SetStatus) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := setStatus.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, setStatus.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.UnitOfWork.CommerceWebsiteRepository().UpdateFields(ctx, setStatus.Id, map[string]any{"status": setStatus.Status}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (SetStatus) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Set commerce website status",
		"Updates the status of a website belonging to the authenticated user's business account.",
		types.HttpRequestTypeUriQuery,
		http.MethodPatch,
		"/v1/commerce/websites/:id/status",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
