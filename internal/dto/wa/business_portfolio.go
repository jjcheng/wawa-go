package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type BusinessPortfolio struct {
	dto.DTOBase
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id"`
	Name                    string `json:"name"`
	New                     bool   `json:"new"` // indicate this is a new business portfolio
}

func NewBusinessPortfolio(businessPortfolio dao_wa.BusinessPortfolio, new bool) BusinessPortfolio {
	return BusinessPortfolio{
		DTOBase: dto.DTOBase{
			Id:            businessPortfolio.Id,
			AddedAt:       businessPortfolio.AddedAt,
			LastUpdatedAt: businessPortfolio.LastUpdatedAt,
		},
		MetaBusinessPortfolioId: businessPortfolio.MetaBusinessPortfolioId,
		Name:                    businessPortfolio.Name,
		New:                     new,
	}
}
