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
	Id   string `json:"id" val:"required" description:"template id"`
	Name string `json:"name" val:"required" description:"name of the template"`
}

func (delete *Delete) Validate() []exception.InputException {
	delete.Id = strings.TrimSpace(delete.Id)
	delete.Name = strings.TrimSpace(delete.Name)
	errors := []exception.InputException{}
	if delete.Id == "" {
		errors = append(errors, exception.NewInputException("id", "missing id"))
	}
	if delete.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	return errors
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to access this WABA")
	}
	metaWABAId := user.WA.BusinessAccount.MetaWABAId
	if err := dependencies.Whatsapp.DeleteTemplate(ctx, metaWABAId, delete.Name, delete.Id, user.WA.BusinessPortfolioAccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error())
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp template",
		"Deletes a WhatsApp message template for an authorized WABA.",
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
