package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type MessageStatus struct {
	dto.DTOBase
	WAMessageId  string                `json:"wa_message_id"`
	Status       types.WAMessageStatus `json:"status"`
	Timestamp    int64                 `json:"timestamp"`
	ErrorMessage string                `json:"error_message"`
	Payload      map[string]any        `json:"-"`
}

func NewMessageStatus(messageStatusEvent dao_wa.MessageStatus) MessageStatus {
	return MessageStatus{
		DTOBase: dto.DTOBase{
			Id:         messageStatusEvent.Id,
			EntryDate:  messageStatusEvent.EntryDate,
			LastUpdate: messageStatusEvent.LastUpdate,
		},
		WAMessageId:  messageStatusEvent.WAMessageId,
		Status:       messageStatusEvent.Status,
		Timestamp:    messageStatusEvent.Timestamp,
		Payload:      messageStatusEvent.Payload,
		ErrorMessage: messageStatusEvent.ErrorMessage,
	}
}
