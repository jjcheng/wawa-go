package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	Sending            bool                  `json:"sending"`
	PhoneNumber        string                `json:"phone_number"`
	PhoneNumberId      string                `json:"phone_number_id"`
	CustomerName       string                `json:"customer_name"`
	CustomerWAId       string                `json:"customer_wa_id"`
	CustomerMetaUserId string                `json:"customer_meta_user_id"`
	WAMessageId        string                `json:"wa_message_id"`
	Timestamp          int64                 `json:"timestamp"`
	Type               string                `json:"type"`
	Status             types.WAMessageStatus `json:"status"`
	Payload            map[string]any        `json:"payload"`
	AttachmentURL      string                `json:"attachment_url,omitempty"`
	// billing
	Billable    bool   `json:"billable"`
	Category    string `json:"category,omitempty"`
	BillingType string `json:"billing_type,omitempty"`
}

func NewMessage(message dao_wa.Message) Message {
	return Message{
		DTOBase: dto.DTOBase{
			Id:         message.Id,
			EntryDate:  message.EntryDate,
			LastUpdate: message.LastUpdate,
		},
		Sending:            message.Sending,
		PhoneNumber:        message.PhoneNumber,
		PhoneNumberId:      message.PhoneNumberId,
		CustomerName:       message.CustomerName,
		CustomerWAId:       message.CustomerWAId,
		CustomerMetaUserId: message.CustomerMetaUserId,
		WAMessageId:        message.WAMessageId,
		Timestamp:          message.Timestamp,
		Type:               message.Type,
		Status:             message.Status,
		Payload:            message.Payload,
		AttachmentURL:      message.AttachmentURL,
		Billable:           message.Billable,
		BillingType:        message.BillingType,
		Category:           message.Category,
	}
}
