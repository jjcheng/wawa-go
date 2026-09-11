package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type PhoneNumber struct {
	dao.DAOBase
	MetaBusinessPortfolioId string `gorm:"column:meta_business_portfolio_id"`
	MetaWABAId              string `gorm:"column:meta_waba_id"`
	MetaPhoneNumberId       string `gorm:"column:meta_phone_number_id"`
	PhoneNumber             string `gorm:"column:phone_number"`
	WAId                    string `gorm:"wa_id"`
	Name                    string `gorm:"column:name"`
	UserId                  int32  `gorm:"column:user_id"`
	// two-step verification PIN set at registration, required to re-register the number later
	RegistrationPin string                    `gorm:"column:registration_pin"`
	Status          types.WAPhoneNumberStatus `gorm:"column:status"`
	// from account.users table
	UserName string `gorm:"column:user_name;->"`
}

func (PhoneNumber) TableName() string {
	return "wa.phone_numbers"
}

func (phoneNumber PhoneNumber) Base() dao.DAOBase {
	return phoneNumber.DAOBase
}
