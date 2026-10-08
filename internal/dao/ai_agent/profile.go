package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/lib/pq"
)

type Profile struct {
	dao.DAOBase
	BusinessAccountId int32            `gorm:"column:business_account_id"`
	Name              string           `gorm:"column:name"`
	Description       string           `gorm:"column:description"`
	BusinessInfo      []map[string]any `gorm:"column:business_info;type:jsonb;serializer:json"`
	BudgetDaily       int32            `gorm:"column:budget_daily"`
	Budget7Days       int32            `gorm:"column:budget_7_days"`
	Budget30Days      int32            `gorm:"column:budget_30_days"`
	HandoverMessage   string           `gorm:"column:handover_message"`
	NeverSayPhrases   pq.StringArray   `gorm:"column:never_say_phrases;type:text[]"`
}

func (Profile) TableName() string {
	return "ai_agent.profiles"
}

func (profile Profile) Base() dao.DAOBase {
	return profile.DAOBase
}
