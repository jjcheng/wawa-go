package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type PhoneNumber struct {
	dao.DAOBase
	BusinessAccountId  int32                     `gorm:"column:business_account_id"` // in case it's not assigned to any user
	MetaPhoneNumberId  string                    `gorm:"column:meta_phone_number_id"`
	DisplayPhoneNumber string                    `gorm:"-"` // not a db column
	WAId               string                    `gorm:"-"` // not a db column
	Name               string                    `gorm:"column:name"`
	Status             types.WAPhoneNumberStatus `gorm:"column:status"`
	MetaAgentId        string                    `gorm:"column:meta_agent_id"`
	AgentEnabled       bool                      `gorm:"column:agent_enabled"` // business agent is setup and ready to handle customers
	// two-step verification PIN set at registration, required to re-register the number later
	RegistrationPin string `gorm:"-"` // not a db column
	// encryption
	DisplayPhoneNumberEncrypted string `gorm:"column:display_phone_number_encrypted"`
	WAIdEncrypted               string `gorm:"column:wa_id_encrypted"`
	WAIdHash                    string `gorm:"column:wa_id_hash"`
	RegistrationPinEncrypted    string `gorm:"column:registration_pin_encrypted"`
}

func (PhoneNumber) TableName() string {
	return "wa.phone_numbers"
}

func (phoneNumber PhoneNumber) Base() dao.DAOBase {
	return phoneNumber.DAOBase
}
