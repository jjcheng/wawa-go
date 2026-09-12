package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type BusinessAccount struct {
	dao.DAOBase
	BussinessPortfolioId int32  `gorm:"column:business_portfolio_id"`
	WABAId               string `gorm:"column:waba_id"`
	Name                 string `gorm:"column:name"`
	TimezoneId           string `gorm:"column:timezone_id"`
	Currency             string `gorm:"column:currency"`
}

func (BusinessAccount) TableName() string {
	return "wa.business_accounts"
}

func (businessAccount BusinessAccount) Base() dao.DAOBase {
	return businessAccount.DAOBase
}
