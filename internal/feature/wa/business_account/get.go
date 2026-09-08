package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"

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
	//MetaWABAId string `uri:"meta_waba_id" description:"meta WABA id"`
}

func (get *Get) Validate() []exception.InputException {
	// var errors []exception.InputException
	// get.MetaWABAId = strings.TrimSpace(get.MetaWABAId)
	// return errors
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusForbidden, "you are not authenticated")
	}
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	// get user's business account
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByUserId(ctx, user.Id)
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
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/wa/business-accounts",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
