package dao_customer

import (
	"database/sql"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipient struct {
	dao.DAOBase
	CustomerName       string                        `gorm:"column:customer_name"`         // denormlize table
	CustomerWAId       string                        `gorm:"column:customer_wa_id"`        // denormlize table
	CustomerMetaUserId string                        `gorm:"column:customer_meta_user_id"` // denormlize table
	CampaignId         int32                         `gorm:"column:campaign_id"`
	CustomerId         int32                         `gorm:"column:customer_id"`
	UserId             int32                         `gorm:"column:user_id"`
	Payload            map[string]any                `gorm:"column:payload;type:jsonb;serializer:json"`
	Status             types.CampaignRecipientStatus `gorm:"column:status"`
	WAMessageId        string                        `gorm:"column:wa_message_id"`
	LastError          string                        `gorm:"column:last_error"`
	Attempts           int32                         `gorm:"column:attempts"`
	NextAttemptAt      sql.NullTime                  `gorm:"column:next_attempt_at"`
}

func (CampaignRecipient) TableName() string {
	return "customer.campaign_recipient"
}

func (campaignRecipient CampaignRecipient) Base() dao.DAOBase {
	return campaignRecipient.DAOBase
}
