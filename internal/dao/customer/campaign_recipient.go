package dao_customer

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipient struct {
	dao.DAOBase
	CampaignId    int32                         `gorm:"column:campaign_id"`
	CustomerId    int32                         `gorm:"column:customer_id"`
	MessageId     *int32                        `gorm:"message_id"`
	Payload       map[string]any                `gorm:"column:payload;type:jsonb;serializer:json"`
	Status        types.CampaignRecipientStatus `gorm:"column:status"`
	LastError     string                        `gorm:"column:last_error"`
	Attempts      int32                         `gorm:"column:attempts"`
	NextAttemptAt *time.Time                    `gorm:"column:next_attempt_at"`
}

func (CampaignRecipient) TableName() string {
	return "customer.campaign_recipient"
}

func (campaignRecipient CampaignRecipient) Base() dao.DAOBase {
	return campaignRecipient.DAOBase
}
