package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dao.DAOBase
	Sending            bool                  `gorm:"column:sending"`
	PhoneNumber        string                `gorm:"column:phone_number"`
	PhoneNumberId      string                `gorm:"column:phone_number_id"`
	CustomerName       string                `gorm:"column:customer_name"`
	CustomerWAId       string                `gorm:"column:customer_wa_id"`
	CustomerMetaUserId string                `gorm:"column:customer_meta_user_id"`
	WAMessageId        string                `gorm:"column:wa_message_id"`
	Timestamp          int64                 `gorm:"column:timestamp"`
	Type               string                `gorm:"column:type"`
	Status             types.WAMessageStatus `gorm:"column:status"`
	Payload            map[string]any        `gorm:"column:payload;type:jsonb;serializer:json"`
	AttachmentURL      string                `gorm:"column:attachment_url"`
	// for billing
	Price    float32 `gorm:"price"`
	Category string  `gorm:"category"`
}

func (Message) TableName() string {
	return "wa.messages"
}

func (message Message) Base() dao.DAOBase {
	return message.DAOBase
}
