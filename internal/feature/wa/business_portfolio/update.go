package feature_wa_business_portfolio

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

type Update struct {
	MetaBusinessPortfolioId string `uri:"meta_business_portfolio_id" val:"required" description:"Meta business portfolio id"`
}

func (update *Update) Validate() []exception.InputException {
	var errors []exception.InputException
	update.MetaBusinessPortfolioId = strings.TrimSpace(update.MetaBusinessPortfolioId)
	if update.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	return errors
}

func (update Update) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	existing, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, update.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessName, err := dependencies.Whatsapp.GetBusinessName(ctx, update.MetaBusinessPortfolioId, existing.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusBadGateway, err.Error())
	}
	if existing.Name != businessName {
		existing.Name = businessName
		if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, existing); err != nil {
			return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	result := dto_wa.NewBusinessPortfolio(*existing, false)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business portfolio",
		"Refreshes and stores the business portfolio name and logo from the WhatsApp API.",
		types.HttpRequestTypeUri,
		http.MethodPatch,
		"/v1/wa/business-portfolios/:meta_business_portfolio_id",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
