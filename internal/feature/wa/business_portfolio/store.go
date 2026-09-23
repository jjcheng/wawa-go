package feature_wa_business_portfolio

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Store struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"return in embedded signup"`
	Name                    string `json:"name" val:"required" description:"portfolio name taken from the WABA owner_business_info"`
	AccessToken             string `json:"access_token" val:"required" description:"access token exchanged from authorization code during embedded signup"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	store.MetaBusinessPortfolioId = strings.TrimSpace(store.MetaBusinessPortfolioId)
	store.Name = strings.TrimSpace(store.Name)
	store.AccessToken = strings.TrimSpace(store.AccessToken)
	if store.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing Meta business portfolio id"))
	}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing business portfolio name"))
	}
	if store.AccessToken == "" {
		errors = append(errors, exception.NewInputException("access_token", "missing access token"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	existing, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, store.MetaBusinessPortfolioId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	businessName := store.Name
	if existing != nil {
		if existing.Name != businessName {
			existing.Name = businessName
			existing.AccessToken = store.AccessToken
			// need to return error here becuase if access token is not stored, everything does not work
			if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, existing); err != nil {
				return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		d := dto_wa.NewBusinessPortfolio(*existing, false)
		return dto.NewSuccessResponse(&d)
	}
	businessPortfolio := dao_wa.BusinessPortfolio{
		MetaBusinessPortfolioId: store.MetaBusinessPortfolioId,
		Name:                    businessName,
		AccessToken:             store.AccessToken,
	}
	if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Insert(ctx, &businessPortfolio); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	d := dto_wa.NewBusinessPortfolio(businessPortfolio, true)
	return dto.NewSuccessResponse(&d)
}
