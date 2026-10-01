package feature_wa_template

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Delete struct {
	Id string `json:"id" val:"required" description:"template id"`
}

func (delete *Delete) Validate() []exception.InputException {
	delete.Id = strings.TrimSpace(delete.Id)
	errors := []exception.InputException{}
	if delete.Id == "" {
		errors = append(errors, exception.NewInputException("id", "missing id"))
	}
	return errors
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	template, err := dependencies.Whatsapp.GetTemplate(ctx, delete.Id, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	if template == nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, "template not found", nil)
	}
	if err := dependencies.Whatsapp.DeleteTemplate(ctx, user.WA.BusinessAccount.WABAId, template.Name, delete.Id, user.WA.BusinessPortfolioAccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp template",
		"Delete a WhatsApp message template by WhatsApp API. Only MASTER user can access this endpoint.",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/v1/wa/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
