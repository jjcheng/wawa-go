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
	UserId              int32                `json:"user_id"`
	RecipientCount      int32                `json:"recipient_count"`
	Status              types.CampaignStatus `json:"status"`
	Token               string               `json:"token"`
	SendTemplatePayload map[string]any       `json:"send_template_payload"`
	TemplatePayload     map[string]any       `json:"template_payload"`
}

func NewCampaign(campaign dao_customer.Campaign) Campaign {
	c := Campaign{
		DTOBase: dto.DTOBase{
			Id:         campaign.Id,
			EntryDate:  campaign.EntryDate,
			LastUpdate: campaign.LastUpdate,
		},
		Name:                campaign.Name,
		WATemplateId:        campaign.WATemplateId,
		UserId:              campaign.UserId,
		RecipientCount:      campaign.RecipientCount,
		Status:              campaign.Status,
		Token:               campaign.Token,
		SendTemplatePayload: campaign.SendTemplatePayload,
		SendDate:            campaign.SendDate,
		TemplatePayload:     campaign.TemplatePayload,
	}
	return c
}
