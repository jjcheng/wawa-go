package dao_account

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dao.DAOBase
	Name              string           `gorm:"column:name"`
	PhoneNumber       string           `gorm:"column:phone_number"`
	Email             string           `gorm:"column:email"`
	PasswordHash      string           `gorm:"column:password_hash"`
	AccessTokenHash   *string          `gorm:"column:access_token_hash"`
	AccessTokenExpiry *time.Time       `gorm:"column:access_token_expiry"`
	Description       string           `gorm:"column:description"`
	Type              types.UserType   `gorm:"column:type"`
	Status            types.UserStatus `gorm:"column:status"`
}

func (User) TableName() string {
	return "account.users"
}

func (user User) Base() dao.DAOBase {
	return user.DAOBase
}
