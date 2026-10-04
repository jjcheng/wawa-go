package dao_wa

import "github.com/jjcheng/wawa-go/internal/dao"

type BusinessAgentSchedule struct {
	dao.DAOBase
	PhoneNumberId int32  `gorm:"column:phone_number_id"`
	Enabled       bool   `gorm:"column:enabled"`
	Cron          string `gorm:"column:cron"`
}

func (BusinessAgentSchedule) TableName() string {
	return "wa.business_agent_schedules"
}

func (businessAgentSchedule BusinessAgentSchedule) Base() dao.DAOBase {
	return businessAgentSchedule.DAOBase
}
