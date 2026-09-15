package dto_customer

import (
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
)

type CampaignRecipient struct {
	dto.DTOBase
	CampaignId int32          `json:"campaign_id"`
	CustomerId int32          `json:"customer_id"`
	MessageId  *int32         `json:"message_id"`
	Payload    map[string]any `json:"-"` // do not include
	// from customers
	CustomerName        string `json:"customer_name"`
	CustomerCountryCode string `json:"customer_country_code"`
	CustomerPhoneNumber string `json:"customer_phone_number"`
	// lazy loaded
	Message *dto_wa.Message `json:"message,omitempty"`
}

func NewCampaignRecipient(campaignRecipient dao_customer.CampaignRecipient) CampaignRecipient {
	cr := CampaignRecipient{
		DTOBase: dto.DTOBase{
			Id:         campaignRecipient.Id,
			EntryDate:  campaignRecipient.EntryDate,
			LastUpdate: campaignRecipient.LastUpdate,
		},
		CampaignId:          campaignRecipient.CampaignId,
		CustomerId:          campaignRecipient.CustomerId,
		MessageId:           campaignRecipient.MessageId,
		CustomerName:        campaignRecipient.CustomerName,
		CustomerCountryCode: campaignRecipient.CustomerCountryCode,
		CustomerPhoneNumber: campaignRecipient.CustomerPhoneNumber,
		Payload:             campaignRecipient.Payload,
	}
	if campaignRecipient.Message != nil {
		message := dto_wa.NewMessage(*campaignRecipient.Message)
		cr.Message = &message
	}
	return cr
}
