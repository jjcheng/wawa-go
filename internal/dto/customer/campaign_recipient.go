package dto_customer

import (
	"database/sql"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipient struct {
	dto.DTOBase
	CustomerName       string                        `json:"customer_name"` // denormlize table
	CustomerWAId       string                        `json:"customer_wa_id"`
	CustomerMetaUserId string                        `json:"customer_meta_user_id"`
	CampaignId         int32                         `json:"campaign_id"`
	CustomerId         int32                         `json:"customer_id"`
	UserId             int32                         `json:"user_id"`
	Payload            map[string]any                `json:"-"` // do not include
	Status             types.CampaignRecipientStatus `json:"status"`
	WAMessageId        string                        `json:"wa_message_id"`
	LastError          string                        `json:"last_error"`
	Attempts           int32                         `json:"attempts"`
	NextAttemptAt      sql.NullTime                  `json:"next_attempt_at"`
}

func NewCampaignRecipient(campaignRecipient dao_customer.CampaignRecipient) CampaignRecipient {
	return CampaignRecipient{
		DTOBase: dto.DTOBase{
			Id:         campaignRecipient.Id,
			EntryDate:  campaignRecipient.EntryDate,
			LastUpdate: campaignRecipient.LastUpdate,
		},
		CustomerName:       campaignRecipient.CustomerName,
		CustomerWAId:       campaignRecipient.CustomerWAId,
		CustomerMetaUserId: campaignRecipient.CustomerMetaUserId,
		CampaignId:         campaignRecipient.CampaignId,
		CustomerId:         campaignRecipient.CustomerId,
		UserId:             campaignRecipient.UserId,
		Payload:            campaignRecipient.Payload,
		Status:             campaignRecipient.Status,
		WAMessageId:        campaignRecipient.WAMessageId,
		LastError:          campaignRecipient.LastError,
		Attempts:           campaignRecipient.Attempts,
		NextAttemptAt:      campaignRecipient.NextAttemptAt,
	}
}
