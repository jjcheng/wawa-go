package feature_wa_business_account

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type UpdateName struct {
	MetaWABAId string `json:"meta_waba_id" val:"required" description:"Meta WABA id"`
}

func (update *UpdateName) Validate() []exception.InputException {
	var errors []exception.InputException
	update.MetaWABAId = strings.TrimSpace(update.MetaWABAId)
	if update.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	return errors
}

func (update UpdateName) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	existing, ex := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, update.MetaWABAId)
	if ex != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](ex.StatusCode, ex.Message)
	}
	wabaName, err := dependencies.Whatsapp.GetWABAName(ctx, update.MetaWABAId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, update.MetaWABAId)
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusBadGateway, "error getting WABA name")
	}
	existing.Name = wabaName
	if err := dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, existing); err != nil {
		dependencies.Logger.ErrorFunction(err, update)
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, "error updating business account")
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
		"/wa/v1/business-accounts/name",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("error getting WABA name", http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException("error updating business account", http.StatusInternalServerError)),
		},
	)
}
