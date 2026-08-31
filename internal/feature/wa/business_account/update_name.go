package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type UpdateName struct {
	MetaWABAId string `json:"meta_waba_id" val:"required" description:"Meta WABA id"`
}

func (update *UpdateName) Validate() []exception.InputException {
	var errors []exception.InputException
	update.MetaWABAId = strings.TrimSpace(update.MetaWABAId)
	if update.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	return errors
}

func (update UpdateName) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	existing, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, update.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business account not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetBusinessPortfolioByWABAId(ctx, update.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	wabaName, err := dependencies.Whatsapp.GetWABAName(ctx, update.MetaWABAId, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	existing.Name = wabaName
	if err := dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, existing); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	result := dto_wa.NewBusinessAccount(*existing)
	return dto.NewSuccessResponse(&result)
}

func (UpdateName) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business account name",
		"Refreshes and stores the WABA name from the WhatsApp API.",
		types.HttpRequestTypeJSON,
		http.MethodPatch,
		"/v1/wa/business-accounts/name",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
