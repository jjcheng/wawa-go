package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type BusinessAccount struct {
	dto.DTOBase
	WABAId     string `json:"waba_id"`
	Name       string `json:"name"`
	TimezoneId string `json:"timezone_id"`
	Currency   string `json:"currency"`
	// used only when user updating WABA
	MetaBusinessPortfolioId   string `json:"meta_business_portfolio_id"`
	MetaBusinessPortfolioName string `json:"meta_business_portfolio_name"`
}

func NewBusinessAccount(businessAccount dao_wa.BusinessAccount, metaBusinessPortfolioId string, metaBusinessPortfolioName string) BusinessAccount {
	return BusinessAccount{
		DTOBase: dto.DTOBase{
			Id:            businessAccount.Id,
			AddedAt:       businessAccount.AddedAt,
			LastUpdatedAt: businessAccount.LastUpdatedAt,
		},
		WABAId:                    businessAccount.WABAId,
		Name:                      businessAccount.Name,
		Currency:                  businessAccount.Currency,
		TimezoneId:                businessAccount.TimezoneId,
		MetaBusinessPortfolioId:   metaBusinessPortfolioId,
		MetaBusinessPortfolioName: metaBusinessPortfolioName,
	}
}
