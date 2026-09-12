package dto_customer

import (
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Campaign struct {
	dto.DTOBase
	Name                string               `json:"name"`
	SendDate            time.Time            `json:"send_date"`
	WATemplateId        string               `json:"wa_template_id"`
	CustomerIds         []int32              `json:"customer_ids"`
	UserId              int32                `json:"user_id"`
	Status              types.CampaignStatus `json:"status"`
	Token               string               `json:"token"`
	SendTemplatePayload map[string]any       `json:"send_template_payload"`
	TemplatePayload     map[string]any       `json:"template_payload"`
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
		Name:                campaign.Name,
		WATemplateId:        campaign.WATemplateId,
		CustomerIds:         []int32(campaign.CustomerIds),
		Status:              campaign.Status,
		Token:               campaign.Token,
		Customers:           customers,
		SendTemplatePayload: campaign.SendTemplatePayload,
		SendDate:            campaign.SendDate,
		TemplatePayload:     campaign.TemplatePayload,
	}
	return c
}
