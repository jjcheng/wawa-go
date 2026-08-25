package feature_wa_business_portfolio

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
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"Meta business portfolio id"`
}

func (update *UpdateName) Validate() []exception.InputException {
	var errors []exception.InputException
	update.MetaBusinessPortfolioId = strings.TrimSpace(update.MetaBusinessPortfolioId)
	if update.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	return errors
}

func (update UpdateName) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	existing, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, update.MetaBusinessPortfolioId)
	if ex != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](ex.StatusCode, ex.Message)
	}
	businessName, err := dependencies.Whatsapp.GetBusinessName(ctx, update.MetaBusinessPortfolioId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, update.MetaBusinessPortfolioId)
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusBadGateway, "error getting business name")
	}
	existing.Name = businessName
	if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, existing); err != nil {
		dependencies.Logger.ErrorFunction(err, update)
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, "error updating business portfolio")
	}
	result := dto_wa.NewBusinessPortfolio(*existing, false)
	return dto.NewSuccessResponse(&result)
}

func (UpdateName) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business portfolio name",
		"Refreshes and stores the business portfolio name from the WhatsApp API.",
		types.HttpRequestTypeJSON,
		http.MethodPatch,
		"/wa/v1/business-portfolios/name",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("error getting business name", http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException("error updating business portfolio", http.StatusInternalServerError)),
		},
	)
}
