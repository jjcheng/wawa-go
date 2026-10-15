package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Skill struct {
	dto.DTOBase
	Type        types.AIAgentSkillType `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	AlwaysApply bool                   `json:"always_apply"`
	Inputs      []SkillInput           `json:"inputs"`
}

type SkillInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func NewSkill(skill dao_ai_agent.Skill, typ types.AIAgentSkillType) Skill {
	s := Skill{
		DTOBase: dto.DTOBase{
			Id:            skill.Id,
			AddedAt:       skill.AddedAt,
			LastUpdatedAt: skill.LastUpdatedAt,
		},
		Type:        typ,
		Title:       skill.Title,
		Description: skill.Description,
		Enabled:     skill.Enabled,
		AlwaysApply: skill.AlwaysApply,
	}
	s.Inputs = helper.Map(skill.Inputs, func(s dao_ai_agent.SkillInput) SkillInput {
		return SkillInput{
			Title:       s.Title,
			Description: s.Description,
			Required:    s.Required,
		}
	})
	return s
}
