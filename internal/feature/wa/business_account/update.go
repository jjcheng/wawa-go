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

type Update struct {
}

func (update *Update) Validate() []exception.InputException {
	return nil
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetById(ctx, user.BusinessAccountId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business account not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	waba, err := dependencies.Whatsapp.GetWABA(ctx, user.WA.BusinessAccount.WABAId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusBadGateway, err.Error(), err)
	}
	businessAccount.Name = waba.Name
	businessAccount.TimezoneId = waba.TimezoneID
	if err := dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, businessAccount); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// now update business portfolio
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetById(ctx, user.BusinessAccountId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business portfolio not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if businessPortfolio.Name != waba.OwnerBusinessInfo.Name {
		businessPortfolio.Name = waba.OwnerBusinessInfo.Name
		if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, businessPortfolio); err != nil {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	result := dto_wa.NewBusinessAccount(*businessAccount, waba.OwnerBusinessInfo.ID, waba.OwnerBusinessInfo.Name)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business account",
		"Refreshes and stores the WABA name from the WhatsApp API. Only MASTER user can access this endpoint",
		types.HttpRequestTypeNone,
		http.MethodPatch,
		"/v1/wa/business-accounts",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
