package dao_cm

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Conversation struct {
	dao.DAOBase
	UserId     int32                    `gorm:"column:user_id"`
	Title      string                   `gorm:"column:title"`      // first rewrite's user_message
	Identifier string                   `gorm:"column:identifier"` // uuid
	Channel    types.ChatMessageChannel `gorm:"column:channel"`    // WHATSAPP, WEB
	Source     string                   `gorm:"column:source"`     // campaign://1
}

func (Conversation) TableName() string {
	return "cm_conversation"
}

func (conversation Conversation) Base() dao.DAOBase {
	return conversation.DAOBase
}
