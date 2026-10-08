package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type FAQ struct {
	dto.DTOBase
	ProfileId int32                `json:"profile_id"`
	Type      types.AIAgentFAQType `json:"type"`
	Question  string               `json:"question"`
	Answer    string               `json:"answer"`
	Enabled   bool                 `json:"enabled"`
}

func NewFAQ(faq dao_ai_agent.FAQ) FAQ {
	return FAQ{
		DTOBase: dto.DTOBase{
			Id:            faq.Id,
			AddedAt:       faq.AddedAt,
			LastUpdatedAt: faq.LastUpdatedAt,
		},
		ProfileId: faq.ProfileId,
		Type:      faq.Type,
		Question:  faq.Question,
		Answer:    faq.Answer,
		Enabled:   faq.Enabled,
	}
}
