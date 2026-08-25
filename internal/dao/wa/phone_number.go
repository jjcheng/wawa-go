package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type PhoneNumber struct {
	dao.DAOBase
	MetaBusinessPortfolioId string `gorm:"column:meta_business_portfolio_id"`
	MetaWABAId              string `gorm:"column:meta_waba_id"`
	MetaPhoneNumberId       string `gorm:"column:meta_phone_number_id"`
	PhoneNumber             string `gorm:"column:phone_number"`
	Name                    string `gorm:"column:name"`
}

func (PhoneNumber) TableName() string {
	return "wa.phone_numbers"
}

func (phoneNumber PhoneNumber) Base() dao.DAOBase {
	return phoneNumber.DAOBase
}
