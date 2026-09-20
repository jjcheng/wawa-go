package dao_customer

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type BroadcastRecipient struct {
	dao.DAOBase
	BroadcastId int32          `gorm:"column:broadcast_id"`
	CustomerId  int32          `gorm:"column:customer_id"`
	MessageId   *int32         `gorm:"column:message_id"`
	Payload     map[string]any `gorm:"-"` // not a db column
	// from customers table
	CustomerName string `gorm:"column:customer_name;->"`
	// from messages table
	Message *dao_wa.Message `gorm:"column:message;->"`
	// encryption
	EncryptionID     string `gorm:"column:encryption_id"`
	PayloadEncrypted string `gorm:"column:payload_encrypted"`
}

func (BroadcastRecipient) TableName() string {
	return "customer.broadcast_recipients"
}

func (broadcastRecipient BroadcastRecipient) Base() dao.DAOBase {
	return broadcastRecipient.DAOBase
}
