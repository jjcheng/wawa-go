package feature_wa_business_portfolio

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
	//MetaBusinessPortfolioId string `uri:"meta_business_portfolio_id"  description:"Meta business portfolio id"`
}

func (get *Get) Validate() []exception.InputException {
	// var errors []exception.InputException
	// get.MetaBusinessPortfolioId = strings.TrimSpace(get.MetaBusinessPortfolioId)
	// return errors
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusForbidden, "you are not authenticated")
	}
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	// if get.MetaBusinessPortfolioId != "" {
	// 	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, get.MetaBusinessPortfolioId)
	// 	if err != nil {
	// 		if errors.Is(err, gorm.ErrRecordNotFound) {
	// 			return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusNotFound, "business portfolio not found")
	// 		}
	// 		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	// 	}
	// 	result := dto_wa.NewBusinessPortfolio(*businessPortfolio, false)
	// 	return dto.NewSuccessResponse(&result)
	// } else {
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	result := dto_wa.NewBusinessPortfolio(*businessPortfolio, false)
	return dto.NewSuccessResponse(&result)
	//}
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp business portfolio",
		"Gets a WhatsApp business portfolio by its Meta business portfolio ID.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/wa/business-portfolios",
		true,
		false,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
