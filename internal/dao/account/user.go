package dao_account

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dao.DAOBase
	Name        string           `gorm:"column:name"`
	CountryCode string           `gorm:"column:country_code"`
	PhoneNumber string           `gorm:"-"` // not a column in db
	Email       string           `gorm:"-"` // not a column in db
	Description string           `gorm:"column:description"`
	Type        types.UserType   `gorm:"column:type"`
	Status      types.UserStatus `gorm:"column:status"`
	// encryption
	EncryptionID         string `gorm:"column:encryption_id"` // used to generate aad
	PhoneNumberEncrypted string `gorm:"column:phone_number_encrypted"`
	PhoneNumberHash      string `gorm:"column:phone_number_hash"`
	PasswordHash         string `gorm:"column:password_hash"`
	EmailEncrypted       string `gorm:"column:email_encrypted"`
	EmailHash            string `gorm:"column:email_hash"`
}

func (User) TableName() string {
	return "account.users"
}

func (user User) Base() dao.DAOBase {
	return user.DAOBase
}
