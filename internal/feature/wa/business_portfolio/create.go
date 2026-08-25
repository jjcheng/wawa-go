package feature_wa_business_portfolio

import (
	"context"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Create struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"return in embedded signup"`
}

func (create *Create) Validate() []exception.InputException {
	var errors []exception.InputException
	create.MetaBusinessPortfolioId = strings.TrimSpace(create.MetaBusinessPortfolioId)
	if create.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	existing, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, create.MetaBusinessPortfolioId)
	if ex != nil && ex.StatusCode != http.StatusNotFound {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](ex.StatusCode, ex.Message)
	}
	businessName, err := dependencies.Whatsapp.GetBusinessName(ctx, create.MetaBusinessPortfolioId)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusBadGateway, "error getting business name")
	}
	if existing != nil {
		if existing.Name != businessName {
			existing.Name = businessName
			if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, existing); err != nil {
				dependencies.Logger.ErrorFunction(err, create)
				return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, "error updating business portfolio")
			}
		}
		d := dto_wa.NewBusinessPortfolio(*existing, false)
		return dto.NewSuccessResponse(&d)
	}
	businessPortfolio := dao_wa.BusinessPortfolio{
		MetaBusinessPortfolioId: create.MetaBusinessPortfolioId,
		Name:                    businessName,
	}
	if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Insert(ctx, &businessPortfolio); err != nil {
		dependencies.Logger.ErrorFunction(err, create)
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, "error creating business portfolio")
	}
	d := dto_wa.NewBusinessPortfolio(businessPortfolio, true)
	return dto.NewSuccessResponse(&d)
}

// func (Create) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Create WhatsApp business portfolio", "Create or refresh a WhatsApp business portfolio from an embedded signup.", types.HttpRequestTypeJSON, "POST", "/wa/v1/business-portfolios", true, false, types.APITagAccount, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("error getting business name", http.StatusBadGateway)),
// 		feature.NewAPIError(*exception.NewCustomException("error creating business portfolio", http.StatusInternalServerError)),
// 		feature.NewAPIError(*exception.NewCustomException("error updating business portfolio", http.StatusInternalServerError)),
// 	})
// }
