package dao_customer

import (
	"database/sql"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Campaign struct {
	dao.DAOBase
	Name         string               `gorm:"column:name"`
	SendDate     sql.NullTime         `gorm:"column:send_date"`
	WATemplateId string               `gorm:"column:wa_template_id"`
	CustomerIds  []int32              `gorm:"column:customer_ids"`
	UserId       int32                `gorm:"column:user_id"`
	Status       types.CampaignStatus `gorm:"column:status"`
}

func (Campaign) TableName() string {
	return "customer.campaigns"
}

func (campaign Campaign) Base() dao.DAOBase {
	return campaign.DAOBase
}
