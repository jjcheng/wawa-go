package dao_customer

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type CampaignRecipient struct {
	dao.DAOBase
	CampaignId int32          `gorm:"column:campaign_id"`
	CustomerId int32          `gorm:"column:customer_id"`
	MessageId  *int32         `gorm:"column:message_id"`
	Payload    map[string]any `gorm:"column:payload;type:jsonb;serializer:json"`
	// from customers table
	CustomerName        string `gorm:"column:customer_name;->"`
	CustomerCountryCode string `gorm:"column:customer_country_code;->"`
	CustomerPhoneNumber string `gorm:"column:customer_phone_number;->"`
	// from messages table
	Message *dao_wa.Message `gorm:"column:message;->"`
}

func (CampaignRecipient) TableName() string {
	return "customer.campaign_recipients"
}

func (campaignRecipient CampaignRecipient) Base() dao.DAOBase {
	return campaignRecipient.DAOBase
}
