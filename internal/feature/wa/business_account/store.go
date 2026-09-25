package feature_wa_business_account

import (
	"context"
	"errors"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// used only in embedded signup
type Store struct {
	BusinessPortfolio *dto_wa.BusinessPortfolio            `json:"business_portfolio" val:"required" description:"business porfolio associated with this business account"`
	WABADetails       *service.WhatsAppWABADetailsResponse `json:"waba_details" val:"required" description:"details of WABA"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	if store.BusinessPortfolio == nil {
		errors = append(errors, exception.NewInputException("business_portfolio", "missing business portfolio"))
	}
	if store.WABADetails == nil {
		errors = append(errors, exception.NewInputException("waba_details", "missing WABA details"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAccount] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAccount](errors)
	}
	existing, _, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetByWABAId(ctx, store.WABADetails.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
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
			BussinessPortfolioId: store.BusinessPortfolio.Id,
			WABAId:               store.WABADetails.ID,
			Name:                 store.WABADetails.Name,
			TimezoneId:           store.WABADetails.TimezoneID,
			Currency:             store.WABADetails.Currency,
		}
		if err := dependencies.UnitOfWork.WABusinessAccountRepository().Insert(ctx, &businessAccount); err != nil {
			return dto.NewFailedResponse[*dto_wa.BusinessAccount](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	d := dto_wa.NewBusinessAccount(businessAccount, "", "")
	return dto.NewSuccessResponse(&d)
}
