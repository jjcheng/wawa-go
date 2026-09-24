package dao_wa

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dao.DAOBase
	Sending       bool                  `gorm:"column:sending"`
	UserId        int32                 `gorm:"phone_number_id"`        // sender
	PhoneNumberId int32                 `gorm:"column:phone_number_id"` // phone used to send
	CustomerId    int32                 `gorm:"column:customer_id"`     // recipient
	WAMessageId   string                `gorm:"column:wa_message_id"`
	Timestamp     int64                 `gorm:"column:timestamp"`
	Type          string                `gorm:"column:type"`
	Status        types.WAMessageStatus `gorm:"column:status"`
	Payload       map[string]any        `gorm:"-"` // not a db column
	AttachmentURL string                `gorm:"column:attachment_url"`
	Token         string                `gorm:"column:token"`
	// for billing
	Billable    bool   `gorm:"column:billable"`
	BillingType string `gorm:"column:billing_type"`
	Category    string `gorm:"column:category"`
	// retry if error
	Attempts      int32      `gorm:"column:attempts"`
	NextAttemptAt *time.Time `gorm:"column:next_attempt_at"`
	ErrorMessage  string     `gorm:"column:error_message"`
	// encryption
	PayloadEncrypted string `gorm:"column:payload_encrypted"`
}

func (Message) TableName() string {
	return "wa.messages"
}

func (message Message) Base() dao.DAOBase {
	return message.DAOBase
}
