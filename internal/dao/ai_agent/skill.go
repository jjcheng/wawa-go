package dao_ai_agent

import "github.com/jjcheng/wawa-go/internal/dao"

type Skill struct {
	dao.DAOBase
	ProfileId   int32  `gorm:"column:profile_id"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	Enabled     bool   `gorm:"column:enabled"`
}

func (Skill) TableName() string {
	return "ai_agent.skills"
}

func (skill Skill) Base() dao.DAOBase {
	return skill.DAOBase
}
