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

type Get struct {
	MetaWABAId string `form:"meta_waba_id" val:"required" description:"meta WABA id"`
}

func (get *Get) Validate() []exception.InputException {
	var errors []exception.InputException
	get.MetaWABAId = strings.TrimSpace(get.MetaWABAId)
	if get.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	return errors
}

func (get Get) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	businessAccount, ex := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, get.MetaWABAId)
	if ex != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](ex.StatusCode, ex.Message)
	}
	result := dto_wa.NewBusinessAccount(*businessAccount)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business account",
		"Gets a WhatsApp business account by its WABA ID.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/wa/v1/business-accounts",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("error getting business account", http.StatusInternalServerError)),
		},
	)
}
