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

type Get struct {
	MetaWABAId string `form:"meta_waba_id" val:"required" description:"meta WABA id"`
}

func (get *Get) Validate() []exception.InputException {
	var errors []exception.InputException
	get.MetaWABAId = strings.TrimSpace(get.MetaWABAId)
	if get.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	return errors
}

func (get Get) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, get.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business account not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
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
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
