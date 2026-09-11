package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type BusinessAccount struct {
	dao.DAOBase
	MetaBusinessPortfolioId string `gorm:"column:meta_business_portfolio_id"`
	MetaWABAId              string `gorm:"column:meta_waba_id"`
	Name                    string `gorm:"column:name"`
	TimeZoneId              string `gorm:"column:time_zone_id"`
}

func (BusinessAccount) TableName() string {
	return "wa.business_accounts"
}

func (businessAccount BusinessAccount) Base() dao.DAOBase {
	return businessAccount.DAOBase
}
