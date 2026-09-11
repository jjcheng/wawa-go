package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type BusinessAccount struct {
	dto.DTOBase
	MetaWABAId string `json:"meta_waba_id"`
	Name       string `json:"name"`
	// used only when updating WABA
	MetaBusinessPortfolioId   string `json:"meta_business_portfolio_id"`
	MetaBusinessPortfolioName string `json:"meta_business_portfolio_name"`
}

func NewBusinessAccount(businessAccount dao_wa.BusinessAccount) BusinessAccount {
	return BusinessAccount{
		DTOBase: dto.DTOBase{
			Id:         businessAccount.Id,
			EntryDate:  businessAccount.EntryDate,
			LastUpdate: businessAccount.LastUpdate,
		},
		MetaWABAId: businessAccount.MetaWABAId,
		Name:       businessAccount.Name,
	}
}
