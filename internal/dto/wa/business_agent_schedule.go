package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type BusinessAgentSchdule struct {
	dto.DTOBase
	PhoneNumberId int32  `json:"phone_number_id"`
	Cron          string `json:"cron"`
	Enabled       bool   `json:"enabled"`
}

func NewBusinessAgentSchdule(businessAgentSchedule dao_wa.BusinessAgentSchedule) BusinessAgentSchdule {
	return BusinessAgentSchdule{
		DTOBase: dto.DTOBase{
			Id:            businessAgentSchedule.Id,
			AddedAt:       businessAgentSchedule.AddedAt,
			LastUpdatedAt: businessAgentSchedule.LastUpdatedAt,
		},
		PhoneNumberId: businessAgentSchedule.PhoneNumberId,
		Cron:          businessAgentSchedule.Cron,
		Enabled:       businessAgentSchedule.Enabled,
	}
}
