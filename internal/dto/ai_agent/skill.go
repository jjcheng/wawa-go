package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Skill struct {
	dto.DTOBase
	Title       string `json:"title"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

func NewSkill(skill dao_ai_agent.Skill) Skill {
	return Skill{
		DTOBase: dto.DTOBase{
			Id:            skill.Id,
			AddedAt:       skill.AddedAt,
			LastUpdatedAt: skill.LastUpdatedAt,
		},
		Title:       skill.Title,
		Description: skill.Description,
		Enabled:     skill.Enabled,
	}
}
