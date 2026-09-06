package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Message struct {
	dto.DTOBase
	Sending             bool           `json:"sending"`
	PhoneNumber         string         `json:"phone_number"`
	PhoneNumberId       string         `json:"phone_number_id"`
	CustomerName        string         `json:"customer_name"`
	CustomerPhoneNumber string         `json:"customer_phone_number"`
	CustomerMetaUserId  string         `json:"customer_meta_user_id"`
	MetaId              string         `json:"meta_id"`
	Timestamp           int64          `json:"timestamp"`
	Type                string         `json:"type"`
	Payload             map[string]any `json:"payload"`
}

func NewMessage(message dao_wa.Message) Message {
	return Message{
		DTOBase: dto.DTOBase{
			Id:         message.Id,
			EntryDate:  message.EntryDate,
			LastUpdate: message.LastUpdate,
		},
		Sending:             message.Sending,
		PhoneNumber:         message.PhoneNumber,
		PhoneNumberId:       message.PhoneNumberId,
		CustomerName:        message.CustomerName,
		CustomerPhoneNumber: message.CustomerPhoneNumber,
		CustomerMetaUserId:  message.CustomerMetaUserId,
		MetaId:              message.MetaId,
		Timestamp:           message.Timestamp,
		Type:                message.Type,
		Payload:             message.Payload,
	}
}
