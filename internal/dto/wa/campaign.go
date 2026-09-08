package dto_wa

import (
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Campaign struct {
	dto.DTOBase
	Name         string                 `json:"name"`
	SendDate     *time.Time             `json:"send_date"`
	WATemplateId string                 `json:"wa_template_id"`
	CustomerIds  []int32                `json:"customer_ids"`
	UserId       int32                  `json:"user_id"`
	Status       types.WACampaignStatus `json:"status"`
	Archived     bool                   `json:"archived"`
	//MessageBase
	// lazy loaded
	Customers []dto_customer.Customer `json:"customers"`
}

func NewCampaign(campaign dao_wa.Campaign, customers []dto_customer.Customer) Campaign {
	c := Campaign{
		DTOBase: dto.DTOBase{
			Id:         campaign.Id,
			EntryDate:  campaign.EntryDate,
			LastUpdate: campaign.LastUpdate,
		},
		//MessageBase:  MessageBase(campaign.MessageBase),
		Name:         campaign.Name,
		WATemplateId: campaign.WATemplateId,
		CustomerIds:  campaign.CustomerIds,
		Status:       campaign.Status,
		Archived:     campaign.Archived,
		Customers:    customers,
	}
	if campaign.SendDate.Valid {
		c.SendDate = &campaign.SendDate.Time
	}
	return c
}
