package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type BusinessAgentKeyword struct {
	dto.DTOBase
	PhoneNumberId     int32  `json:"phone_number_id"`
	Keyword           string `json:"keyword"`
	NotificationTitle string `json:"notification_title"`
}

func NewBusinessAgentKeyword(businessAgentKeyword dao_wa.BusinessAgentKeyword) BusinessAgentKeyword {
	return BusinessAgentKeyword{
		DTOBase: dto.DTOBase{
			Id:            businessAgentKeyword.Id,
			AddedAt:       businessAgentKeyword.AddedAt,
			LastUpdatedAt: businessAgentKeyword.LastUpdatedAt,
		},
		PhoneNumberId:     businessAgentKeyword.PhoneNumberId,
		Keyword:           businessAgentKeyword.Keyword,
		NotificationTitle: businessAgentKeyword.NotificationTitle,
	}
}
