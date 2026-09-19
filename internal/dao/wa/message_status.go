package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

// not linked to a phone number id
type MessageStatus struct {
	dao.DAOBase
	MessageId    int32                 `gorm:"column:message_id"`
	WAMessageId  string                `gorm:"column:wa_message_id"`
	Status       types.WAMessageStatus `gorm:"column:status"`
	Timestamp    int64                 `gorm:"column:timestamp"`
	ErrorMessage string                `gorm:"column:error_message"`
	Payload      map[string]any        `gorm:"-"` // not a column in db
	// encryption
	EncryptionID     string `gorm:"column:encryption_id"`
	PayloadEncrypted string `gorm:"column:payload_encrypted"`
}

func (MessageStatus) TableName() string {
	return "wa.message_status"
}

func (messageStatus MessageStatus) Base() dao.DAOBase {
	return messageStatus.DAOBase
}
