package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Store struct {
	MetaBusinessProtfolioId string `json:"meta_business_portfolio_id" val:"required" description:"returned in embedded signup"`
	MetaWABAId              string `json:"meta_waba_id" val:"required" description:"returned in embedded signup"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	store.MetaBusinessProtfolioId = strings.TrimSpace(store.MetaBusinessProtfolioId)
	store.MetaWABAId = strings.TrimSpace(store.MetaWABAId)
	if store.MetaBusinessProtfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing Meta business portfolio id"))
	}
	if store.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	existing, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, store.MetaWABAId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, err.Error())
		}
	}
	// get waba name
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, store.MetaBusinessProtfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	wabaName, err := dependencies.Whatsapp.GetWABAName(ctx, store.MetaWABAId, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	var businessAccount dao_wa.BusinessAccount
	if existing != nil {
		businessAccount = *existing
		if businessAccount.Name != wabaName {
			businessAccount.Name = wabaName
			// ignore error if any
			_ = dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, &businessAccount)
		}
	} else {
		businessAccount = dao_wa.BusinessAccount{
			MetaBusinessPortfolioId: store.MetaBusinessProtfolioId,
			MetaWABAId:              store.MetaWABAId,
			Name:                    wabaName,
		}
		if err := dependencies.UnitOfWork.WABusinessAccountRepository().Insert(ctx, &businessAccount); err != nil {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	d := dto_wa.NewBusinessAccount(businessAccount)
	return dto.NewSuccessResponse(&d)
}
