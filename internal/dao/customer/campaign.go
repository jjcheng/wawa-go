package dao_customer

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Campaign struct {
	dao.DAOBase
	Name                string               `gorm:"column:name"`
	SendDate            time.Time            `gorm:"column:send_date"`
	WATemplateId        string               `gorm:"column:wa_template_id"`
	UserId              int32                `gorm:"column:user_id"`
	RecipientCount      int32                `gorm:"column:recipient_count"`
	Status              types.CampaignStatus `gorm:"column:status"`
	Token               string               `gorm:"column:token"`          // used to identify the campaign
	AttachmentURL       string               `gorm:"column:attachment_url"` // delete file when campaign is deleted
	SendTemplatePayload map[string]any       `gorm:"column:send_template_payload;type:jsonb;serializer:json"`
	TemplatePayload     map[string]any       `gorm:"column:template_payload;type:jsonb;serializer:json"`
}

func (Campaign) TableName() string {
	return "customer.campaigns"
}

func (campaign Campaign) Base() dao.DAOBase {
	return campaign.DAOBase
}
