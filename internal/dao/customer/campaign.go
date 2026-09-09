package dao_customer

import (
	"database/sql"
	"database/sql/driver"

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
	Name         string               `gorm:"column:name"`
	SendDate     sql.NullTime         `gorm:"column:send_date"`
	WATemplateId string               `gorm:"column:wa_template_id"`
	CustomerIds  CustomerIDs          `gorm:"column:customer_ids;type:integer[]"`
	UserId       int32                `gorm:"column:user_id"`
	Status       types.CampaignStatus `gorm:"column:status"`
	Payload      map[string]any       `gorm:"column:payload;type:jsonb;serializer:json"`
}

func (Campaign) TableName() string {
	return "customer.campaigns"
}

func (campaign Campaign) Base() dao.DAOBase {
	return campaign.DAOBase
}
