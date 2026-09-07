package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dao.DAOBase
	Sending             bool                  `gorm:"column:sending"`
	PhoneNumber         string                `gorm:"column:phone_number"`
	PhoneNumberId       string                `gorm:"column:phone_number_id"`
	CustomerName        string                `gorm:"column:customer_name"`
	CustomerPhoneNumber string                `gorm:"column:customer_phone_number"`
	CustomerMetaUserId  string                `gorm:"column:customer_meta_user_id"`
	WAMessageId         string                `gorm:"column:wa_message_id"`
	Timestamp           int64                 `gorm:"column:timestamp"`
	Type                string                `gorm:"column:type"`
	Status              types.WAMessageStatus `gorm:"column:status"`
	Payload             map[string]any        `gorm:"column:payload;type:jsonb;serializer:json"`
}

func (Message) TableName() string {
	return "wa.messages"
}

func (message Message) Base() dao.DAOBase {
	return message.DAOBase
}
