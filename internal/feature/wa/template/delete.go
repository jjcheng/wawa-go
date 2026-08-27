package feature_wa_template

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Delete struct {
	MetaWABAId string `json:"meta_waba_id" description:"Meta WABA ID"`
	Name       string `json:"name" description:"template name"`
	ID         string `json:"id" description:"template ID"`
}

func (delete *Delete) Validate() []exception.InputException {
	delete.MetaWABAId = strings.TrimSpace(delete.MetaWABAId)
	delete.Name = strings.TrimSpace(delete.Name)
	delete.ID = strings.TrimSpace(delete.ID)
	errors := []exception.InputException{}
	if delete.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	if delete.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing template name"))
	}
	if delete.ID == "" {
		errors = append(errors, exception.NewInputException("id", "missing template id"))
	}
	return errors
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if errors := delete.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	phoneNumbers, ex := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	authorized := false
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaWABAId == delete.MetaWABAId {
			authorized = true
			break
		}
	}
	if !authorized {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to access this WABA")
	}
	businessPortfolio, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, phoneNumbers[0].MetaBusinessPortfolioId)
	if ex != nil {
		if errors.Is(ex, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := dependencies.Whatsapp.DeleteTemplate(ctx, delete.MetaWABAId, delete.Name, delete.ID, businessPortfolio.AccessToken); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp template",
		"Deletes a WhatsApp message template for an authorized WABA.",
		types.HttpRequestTypeJSON,
		http.MethodDelete,
		"/wa/v1/templates",
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
