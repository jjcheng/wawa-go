package dto_customer

import (
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Campaign struct {
	dto.DTOBase
	Name         string               `json:"name"`
	SendDate     *time.Time           `json:"send_date"`
	WATemplateId string               `json:"wa_template_id"`
	CustomerIds  []int32              `json:"customer_ids"`
	UserId       int32                `json:"user_id"`
	Status       types.CampaignStatus `json:"status"`
	// lazy loaded
	Customers []Customer `json:"customers"`
}

func NewCampaign(campaign dao_customer.Campaign, customers []Customer) Campaign {
	c := Campaign{
		DTOBase: dto.DTOBase{
			Id:         campaign.Id,
			EntryDate:  campaign.EntryDate,
			LastUpdate: campaign.LastUpdate,
		},
		Name:         campaign.Name,
		WATemplateId: campaign.WATemplateId,
		CustomerIds:  campaign.CustomerIds,
		Status:       campaign.Status,
		Customers:    customers,
	}
	if campaign.SendDate.Valid {
		c.SendDate = &campaign.SendDate.Time
	}
	return c
}
