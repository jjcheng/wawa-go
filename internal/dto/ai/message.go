package dto_ai

import (
	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	Role           types.AIMessageRole `json:"role"`
	Content        string              `json:"content"`
	ConversationId int32               `json:"conversation_id"`
}

func NewMessage(message dao_ai.Message) Message {
	return Message{
		DTOBase: dto.DTOBase{
			Id:            message.Id,
			AddedAt:       message.AddedAt,
			LastUpdatedAt: message.LastUpdatedAt,
		},
		ConversationId: message.ConversationId,
		Role:           message.Role,
		Content:        message.Content,
	}
}
