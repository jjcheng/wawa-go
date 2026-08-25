package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

// each user can manage multiple phone numbers
type UserPhoneNumber struct {
	dao.DAOBase
	UserId        int32 `gorm:"column:user_id"`
	PhoneNumberId int32 `gorm:"column:phone_number_id"`
}

func (UserPhoneNumber) TableName() string {
	return "wa.user_phone_numbers"
}

func (userPhoneNumber UserPhoneNumber) Base() dao.DAOBase {
	return userPhoneNumber.DAOBase
}
