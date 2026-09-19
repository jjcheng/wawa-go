package dao_customer

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
)

type Customer struct {
	dao.DAOBase
	UserId              int32                `gorm:"column:user_id"`
	DisplayName         string               `gorm:"column:display_name"`    // name tracked by user
	WADisplayName       string               `gorm:"column:wa_display_name"` // name given by WhatsApp
	CountryCode         string               `gorm:"column:country_code"`
	PhoneNumber         string               `gorm:"column:phone_number"`
	MetaUserId          string               `gorm:"column:meta_user_id"`
	WAId                string               `gorm:"column:wa_id"`
	Tags                pq.StringArray       `gorm:"column:tags;type:text[]"`
	Status              types.CustomerStatus `gorm:"column:status"`
	Remarks             string               `gorm:"column:remarks"`
	AdditionalData      map[string]any       `gorm:"column:additional_data;type:jsonb;serializer:json"`
	ImportedPhoneNumber string               `gorm:"column:imported_phone_number"` // to prevent duplicate when importing
	Token               string               `gorm:"column:token"`                 // uuid to identify customer
	// encryption
	PhoneNumberEncrypted         string `gorm:"column:phone_number_encrypted"`
	PhoneNumberHash              string `gorm:"column:phone_number_hash"`
	WAIdEncrypted                string `gorm:"column:wa_id_encrypted"`
	WAIdHash                     string `gorm:"column:wa_id_hash"`
	ImportedPhoneNumberEncrypted string `gorm:"column:imported_phone_number_encrypted"`
	ImportedPhoneNumberHash      string `gorm:"column:imported_phone_number_hash"`
	AdditionalDataEncrypted      string `gorm:"column:additional_data_encrypted"`
}

func (Customer) TableName() string {
	return "customer.customers"
}

func (customer Customer) Base() dao.DAOBase {
	return customer.DAOBase
}
