package dto_customer

import (
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
)

type BroadcastRecipient struct {
	dto.DTOBase
	BroadcastId int32          `json:"broadcast_id"`
	CustomerId  int32          `json:"customer_id"`
	MessageId   *int32         `json:"message_id"`
	Payload     map[string]any `json:"-"` // do not include
	// from customers
	CustomerName        string `json:"customer_name"`
	CustomerCountryCode string `json:"customer_country_code"`
	CustomerPhoneNumber string `json:"customer_phone_number"`
	// lazy loaded
	Message *dto_wa.Message `json:"message,omitempty"`
}

func NewBroadcastRecipient(broadcastRecipient dao_customer.BroadcastRecipient) BroadcastRecipient {
	cr := BroadcastRecipient{
		DTOBase: dto.DTOBase{
			Id:         broadcastRecipient.Id,
			EntryDate:  broadcastRecipient.EntryDate,
			LastUpdate: broadcastRecipient.LastUpdate,
		},
		BroadcastId:  broadcastRecipient.BroadcastId,
		CustomerId:   broadcastRecipient.CustomerId,
		MessageId:    broadcastRecipient.MessageId,
		CustomerName: broadcastRecipient.CustomerName,
		Payload:      broadcastRecipient.Payload,
	}
	if broadcastRecipient.Message != nil {
		message := dto_wa.NewMessage(*broadcastRecipient.Message)
		cr.Message = &message
	}
	return cr
}
