package feature_wa_business_account

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Get struct {
}

func (get *Get) Validate() []exception.InputException {
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	result := *user.WA.BusinessAccount
	result.MetaBusinessPortfolioId = user.WA.BusinessPortfolio.MetaBusinessPortfolioId
	result.MetaBusinessPortfolioName = user.WA.BusinessPortfolio.Name
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
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
