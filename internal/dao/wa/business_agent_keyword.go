package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type BusinessAgentKeyword struct {
	dao.DAOBase
	PhoneNumberId     int32  `gorm:"column:phone_number_id"`
	Keyword           string `gorm:"column:keyword"`
	NotificationTitle string `gorm:"column:notification_title"`
}

func (BusinessAgentKeyword) TableName() string {
	return "wa.business_agent_keywords"
}

func (businessAgentKeyword BusinessAgentKeyword) Base() dao.DAOBase {
	return businessAgentKeyword.DAOBase
}
