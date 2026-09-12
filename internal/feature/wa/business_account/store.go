package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// used only in embedded signup
type Store struct {
	BusinessProtfolioId int32                                `json:"business_portfolio_id" val:"required" description:"id of business portfolio"`
	WABADetails         *service.WhatsAppWABADetailsResponse `json:"waba_details" val:"required" description:"details of WABA"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	if store.BusinessProtfolioId <= 0 {
		errors = append(errors, exception.NewInputException("business_portfolio_id", "missing business portfolio id"))
	}
	if store.WABADetails == nil {
		errors = append(errors, exception.NewInputException("waba_details", "missing WABA details"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	existing, businessPortfolio, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, store.WABADetails.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, err.Error())
		}
	}
	var businessAccount dao_wa.BusinessAccount
	if existing != nil {
		businessAccount = *existing
		if businessAccount.Name != store.WABADetails.Name {
			businessAccount.Name = store.WABADetails.Name
			// ignore errors
			_ = dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, &businessAccount)
		}
	} else {
		businessAccount = dao_wa.BusinessAccount{
			BussinessPortfolioId: businessPortfolio.Id,
			WABAId:               store.WABADetails.ID,
			Name:                 store.WABADetails.Name,
			TimezoneId:           store.WABADetails.TimezoneID,
			Currency:             store.WABADetails.Currency,
		}
		if err := dependencies.UnitOfWork.WABusinessAccountRepository().Insert(ctx, &businessAccount); err != nil {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	d := dto_wa.NewBusinessAccount(businessAccount)
	return dto.NewSuccessResponse(&d)
}
