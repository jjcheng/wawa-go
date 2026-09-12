package dto_customer

import (
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipient struct {
	dto.DTOBase
	CampaignId    int32                         `json:"campaign_id"`
	CustomerId    int32                         `json:"customer_id"`
	MessageId     *int32                        `json:"message_id"`
	Payload       map[string]any                `json:"-"` // do not include
	Status        types.CampaignRecipientStatus `json:"status"`
	LastError     string                        `json:"last_error"`
	Attempts      int32                         `json:"attempts"`
	NextAttemptAt *time.Time                    `json:"next_attempt_at"`
}

func NewCampaignRecipient(campaignRecipient dao_customer.CampaignRecipient) CampaignRecipient {
	return CampaignRecipient{
		DTOBase: dto.DTOBase{
			Id:         campaignRecipient.Id,
			EntryDate:  campaignRecipient.EntryDate,
			LastUpdate: campaignRecipient.LastUpdate,
		},
		MessageId:     campaignRecipient.MessageId,
		CampaignId:    campaignRecipient.CampaignId,
		CustomerId:    campaignRecipient.CustomerId,
		Payload:       campaignRecipient.Payload,
		Status:        campaignRecipient.Status,
		LastError:     campaignRecipient.LastError,
		Attempts:      campaignRecipient.Attempts,
		NextAttemptAt: campaignRecipient.NextAttemptAt,
	}
}
