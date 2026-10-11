package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Profile struct {
	dto.DTOBase
	BusinessAccountId int32              `json:"business_account_id"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	BusinessInfo      []BusinessInfoItem `json:"business_info"`
	BudgetDaily       int32              `json:"budget_daily"`
	Budget7Days       int32              `json:"budget_7_days"`
	Budget30Days      int32              `json:"budget_30_days"`
	UTCOffsetHours    int32              `json:"utc_offset_hours"`
	HandoverMessage   string             `json:"handover_message"`
	NeverSayPhrases   []string           `json:"never_say_phrases"`
}

func NewProfile(profile dao_ai_agent.Profile) Profile {
	d := Profile{
		DTOBase: dto.DTOBase{
			Id:            profile.Id,
			AddedAt:       profile.AddedAt,
			LastUpdatedAt: profile.LastUpdatedAt,
		},
		BusinessAccountId: profile.BusinessAccountId,
		Name:              profile.Name,
		Description:       profile.Description,
		BudgetDaily:       profile.BudgetDaily,
		Budget7Days:       profile.Budget7Days,
		Budget30Days:      profile.Budget30Days,
		UTCOffsetHours:    profile.UTCOffsetHours,
		HandoverMessage:   profile.HandoverMessage,
		NeverSayPhrases:   profile.NeverSayPhrases,
	}
	for _, item := range profile.BusinessInfo {
		businessInfoItem := BusinessInfoItem{}
		if title, ok := item["title"].(string); ok {
			businessInfoItem.Title = title
		}
		if description, ok := item["description"].(string); ok {
			businessInfoItem.Description = description
		}
		d.BusinessInfo = append(d.BusinessInfo, businessInfoItem)
	}
	return d
}
