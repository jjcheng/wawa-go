package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type FAQ struct {
	dao.DAOBase
	ProfileId int32                `gorm:"column:profile_id"`
	Type      types.AIAgentFAQType `gorm:"column:type"`
	Question  string               `gorm:"column:question"`
	Answer    string               `gorm:"column:answer"`
	Enabled   bool                 `gorm:"column:enabled"`
}

func (FAQ) TableName() string {
	return "ai_agent.faqs"
}

func (faq FAQ) Base() dao.DAOBase {
	return faq.DAOBase
}
