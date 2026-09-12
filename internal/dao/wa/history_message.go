package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type HistoryMessage struct {
	dao.DAOBase
	WABAId            string `gorm:"column:waba_id"`
	MetaPhoneNumberId string `gorm:"column:meta_phone_number_id"`
	From              string `gorm:"column:from"`
	WAMessageId       string `gorm:"column:wa_message_id"`
	MessageType       string `gorm:"column:message_type"`
	TextBody          string `gorm:"column:text_body"`
	Timestamp         string `gorm:"column:timestamp"`
	Payload           string `gorm:"column:payload;type:jsonb"`
}

func (HistoryMessage) TableName() string { return "wa.history_messages" }

func (message HistoryMessage) Base() dao.DAOBase { return message.DAOBase }
