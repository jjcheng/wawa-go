package dao_account

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

// 1 user can manage multiple phone numbers
type UserPhoneNumber struct {
	dao.DAOBase
	UserId        int32 `gorm:"column:user_id"`
	PhoneNumberId int32 `gorm:"column:phone_number_id"`
	// join table
	User        *User               `gorm:"foreignKey:UserId;references:Id"`
	PhoneNumber *dao_wa.PhoneNumber `gorm:"foreignKey:PhoneNumberId;references:Id"`
	// for cleaning up
	Processed bool `gorm:"-"`
}

func (UserPhoneNumber) TableName() string {
	return "account.user_phone_numbers"
}

func (userPhoneNumber UserPhoneNumber) Base() dao.DAOBase {
	return userPhoneNumber.DAOBase
}
