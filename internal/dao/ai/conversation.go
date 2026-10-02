package dao_ai

import "github.com/jjcheng/wawa-go/internal/dao"

type Conversation struct {
	dao.DAOBase
	UserId int32  `gorm:"column:user_id"`
	Title  string `gorm:"column:title"`
}

func (Conversation) TableName() string {
	return "ai.conversations"
}

func (conversation Conversation) Base() dao.DAOBase {
	return conversation.DAOBase
}
