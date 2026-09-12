package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	Sending       bool                  `json:"sending"`
	PhoneNumberId int32                 `json:"phone_number_id"`
	CustomerId    int32                 `json:"customer_id"`
	WAMessageId   string                `json:"wa_message_id"`
	Timestamp     int64                 `json:"timestamp"`
	Type          string                `json:"type"`
	Status        types.WAMessageStatus `json:"status"`
	Payload       map[string]any        `json:"payload"`
	CampaignId    *int32                `json:"campaign_id"`
	AttachmentURL string                `json:"attachment_url,omitempty"`
	Billable      bool                  `json:"billable"`
	Category      string                `json:"category,omitempty"`
	BillingType   string                `json:"billing_type,omitempty"`
	// lazy loaded
	PreviewHTML     string `json:"preview_html,omitempty"`
	PreviewDarkHTML string `json:"preview_dark_html,omitempty"`
}

func NewMessage(message dao_wa.Message) Message {
	d := Message{
		DTOBase: dto.DTOBase{
			Id:         message.Id,
			EntryDate:  message.EntryDate,
			LastUpdate: message.LastUpdate,
		},
		Sending:       message.Sending,
		PhoneNumberId: message.PhoneNumberId,
		CustomerId:    message.CustomerId,
		WAMessageId:   message.WAMessageId,
		Timestamp:     message.Timestamp,
		Type:          message.Type,
		Status:        message.Status,
		Payload:       message.Payload,
		AttachmentURL: message.AttachmentURL,
		Billable:      message.Billable,
		BillingType:   message.BillingType,
		Category:      message.Category,
		CampaignId:    message.CampaignId,
	}
	return d
}
