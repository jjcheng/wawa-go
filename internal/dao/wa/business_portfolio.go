package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type BusinessPortfolio struct {
	dao.DAOBase
	MetaBusinessPortfolioId string `gorm:"column:meta_business_portfolio_id"`
	Name                    string `gorm:"column:name"`
	AccessToken             string `gorm:"column:access_token"`
}

func (BusinessPortfolio) TableName() string {
	return "wa.business_portfolios"
}

func (businessPortfolio BusinessPortfolio) Base() dao.DAOBase {
	return businessPortfolio.DAOBase
}
