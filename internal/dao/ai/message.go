package dao_ai

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dao.DAOBase
	ConversationId int32               `gorm:"column:conversation_id"`
	Content        string              `gorm:"column:content"`
	Role           types.AIMessageRole `gorm:"column:role"`
}

func (Message) TableName() string {
	return "ai.messages"
}

func (message Message) Base() dao.DAOBase {
	return message.DAOBase
}
