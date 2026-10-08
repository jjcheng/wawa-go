package dao_ai_worker

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
)

type Message struct {
	dao.DAOBase
	Feature        string                    `gorm:"column:feature"`
	ConversationId int32                     `gorm:"column:conversation_id"`
	Parts          pq.StringArray            `gorm:"column:parts;type:text[]"`
	Role           types.AIWorkerMessageRole `gorm:"column:role"`
	URL            string                    `gorm:"column:url"`
}

func (Message) TableName() string {
	return "ai_worker.messages"
}

func (message Message) Base() dao.DAOBase {
	return message.DAOBase
}
