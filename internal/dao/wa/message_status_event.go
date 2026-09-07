package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type MessageStatusEvent struct {
	dao.DAOBase
	WAMessageId string                `gorm:"column:wa_message_id"`
	Status      types.WAMessageStatus `gorm:"column:status"`
	Timestamp   int64                 `gorm:"column:timestamp"`
	Payload     map[string]any        `gorm:"column:payload;type:jsonb;serializer:json"`
}

func (MessageStatusEvent) TableName() string {
	return "wa.message_status_events"
}

func (messageStatusEvent MessageStatusEvent) Base() dao.DAOBase {
	return messageStatusEvent.DAOBase
}
