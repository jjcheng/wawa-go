package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type MessageStatusEvent struct {
	dto.DTOBase
	WAMessageId string                `json:"wa_message_id"`
	Status      types.WAMessageStatus `json:"status"`
	Timestamp   int64                 `json:"timestamp"`
	Payload     map[string]any        `json:"-"`
}

func NewMessageStatusEvent(messageStatusEvent dao_wa.MessageStatusEvent) MessageStatusEvent {
	return MessageStatusEvent{
		DTOBase: dto.DTOBase{
			Id:         messageStatusEvent.Id,
			EntryDate:  messageStatusEvent.EntryDate,
			LastUpdate: messageStatusEvent.LastUpdate,
		},
		WAMessageId: messageStatusEvent.WAMessageId,
		Status:      messageStatusEvent.Status,
		Timestamp:   messageStatusEvent.Timestamp,
		Payload:     messageStatusEvent.Payload,
	}
}
