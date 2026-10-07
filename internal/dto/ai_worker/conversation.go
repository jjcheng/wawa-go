package dto_ai_worker

import (
	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Conversation struct {
	dto.DTOBase
	UserId int32  `json:"user_id"`
	Title  string `json:"title"`
}

func NewConversation(conversation dao_ai_worker.Conversation) Conversation {
	return Conversation{
		DTOBase: dto.DTOBase{
			Id:            conversation.Id,
			AddedAt:       conversation.AddedAt,
			LastUpdatedAt: conversation.LastUpdatedAt,
		},
		Title: conversation.Title,
	}
}
