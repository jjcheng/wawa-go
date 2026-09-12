package dao_customer

import (
	"database/sql/driver"
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
)

type CustomerIDs []int32

func (customerIDs CustomerIDs) Value() (driver.Value, error) {
	values := make(pq.Int64Array, len(customerIDs))
	for index, customerID := range customerIDs {
		values[index] = int64(customerID)
	}
	return values.Value()
}

func (customerIDs *CustomerIDs) Scan(source any) error {
	var values pq.Int64Array
	if err := values.Scan(source); err != nil {
		return err
	}
	*customerIDs = make(CustomerIDs, len(values))
	for index, customerID := range values {
		(*customerIDs)[index] = int32(customerID)
	}
	return nil
}

type Campaign struct {
	dao.DAOBase
	Name                string               `gorm:"column:name"`
	SendDate            time.Time            `gorm:"column:send_date"`
	WATemplateId        string               `gorm:"column:wa_template_id"`
	CustomerIds         CustomerIDs          `gorm:"column:customer_ids;type:integer[]"`
	UserId              int32                `gorm:"column:user_id"`
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
