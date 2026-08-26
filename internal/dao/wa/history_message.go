package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type HistoryMessage struct {
	dao.DAOBase
	WABAId           string `gorm:"column:waba_id"`
	PhoneNumberId    string `gorm:"column:phone_number_id"`
	From             string `gorm:"column:from"`
	MessageId        string `gorm:"column:message_id"`
	MessageType      string `gorm:"column:message_type"`
	TextBody         string `gorm:"column:text_body"`
	MessageTimestamp string `gorm:"column:message_timestamp"`
	RawPayload       string `gorm:"column:raw_payload;type:jsonb"`
}

func (HistoryMessage) TableName() string { return "wa.history_messages" }

func (message HistoryMessage) Base() dao.DAOBase { return message.DAOBase }
