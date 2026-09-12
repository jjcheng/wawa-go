package dao_account

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dao.DAOBase
	Name         string           `gorm:"column:name"`
	CountryCode  string           `gorm:"country_code"`
	PhoneNumber  string           `gorm:"column:phone_number"`
	PasswordHash string           `gorm:"column:password_hash"`
	Email        string           `gorm:"column:email"`
	Description  string           `gorm:"column:description"`
	Type         types.UserType   `gorm:"column:type"`
	Status       types.UserStatus `gorm:"column:status"`
}

func (User) TableName() string {
	return "account.users"
}

func (user User) Base() dao.DAOBase {
	return user.DAOBase
}
