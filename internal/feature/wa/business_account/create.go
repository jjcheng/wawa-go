package feature_wa_business_account

import (
	"context"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Create struct {
	MetaBusinessProtfolioId string `json:"meta_business_portfolio_id" val:"required" description:"returned in embedded signup"`
	MetaWABAId              string `json:"meta_waba_id" val:"required" description:"returned in embedded signup"`
}

func (create *Create) Validate() []exception.InputException {
	var errors []exception.InputException
	create.MetaBusinessProtfolioId = strings.TrimSpace(create.MetaBusinessProtfolioId)
	create.MetaWABAId = strings.TrimSpace(create.MetaWABAId)
	if create.MetaBusinessProtfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	if create.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dao_wa.BusinessAccount] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dao_wa.BusinessAccount](errors)
	}
	existing, ex := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, create.MetaWABAId)
	if ex != nil && ex.StatusCode != http.StatusNotFound {
		return dto.NewFailedResponse[*dao_wa.BusinessAccount](ex.StatusCode, ex.Message)
	}
	// get waba name
	wabaName, err := dependencies.Whatsapp.GetWABAName(ctx, create.MetaWABAId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, create.MetaWABAId)
		return dto.NewFailedResponse[*dao_wa.BusinessAccount](http.StatusBadGateway, "error getting WABA name")
	}
	var businessAccount dao_wa.BusinessAccount
	if existing != nil {
		businessAccount = *existing
		if businessAccount.Name != wabaName {
			businessAccount.Name = wabaName
			if err := dependencies.UnitOfWork.WABusinessAccountRepository().Update(ctx, &businessAccount); err != nil {
				dependencies.Logger.ErrorFunction(err, businessAccount)
				return dto.NewFailedResponse[*dao_wa.BusinessAccount](http.StatusInternalServerError, "error updating business account")
			}
		}
	} else {
		businessAccount = dao_wa.BusinessAccount{
			MetaBusinessPortfolioId: create.MetaBusinessProtfolioId,
			MetaWABAId:              create.MetaWABAId,
			Name:                    wabaName,
		}
		if err := dependencies.UnitOfWork.WABusinessAccountRepository().Insert(ctx, &businessAccount); err != nil {
			dependencies.Logger.ErrorFunction(err, create)
			return dto.NewFailedResponse[*dao_wa.BusinessAccount](http.StatusInternalServerError, "error creating business account")
		}
	}
	return dto.NewSuccessResponse(&businessAccount)
}
