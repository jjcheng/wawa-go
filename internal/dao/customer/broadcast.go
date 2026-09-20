package dao_customer

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Broadcast struct {
	dao.DAOBase
	Name                string                `gorm:"column:name"`
	SendDate            time.Time             `gorm:"column:send_date"`
	WATemplateId        string                `gorm:"column:wa_template_id"`
	UserId              int32                 `gorm:"column:user_id"`
	RecipientCount      int32                 `gorm:"column:recipient_count"`
	Status              types.BroadcastStatus `gorm:"column:status"`
	Token               string                `gorm:"column:token"` // used to identify the broadcast
	ErrorMessage        string                `gorm:"column:error_message"`
	AttachmentURL       string                `gorm:"column:attachment_url"` // delete file when broadcast is deleted
	SendTemplatePayload map[string]any        `gorm:"column:send_template_payload;type:jsonb;serializer:json"`
	TemplatePayload     map[string]any        `gorm:"column:template_payload;type:jsonb;serializer:json"`
}

func (Broadcast) TableName() string {
	return "customer.broadcasts"
}

func (broadcast Broadcast) Base() dao.DAOBase {
	return broadcast.DAOBase
}
