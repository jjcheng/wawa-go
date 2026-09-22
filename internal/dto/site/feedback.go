package dto_site

import (
	dao_site "github.com/jjcheng/wawa-go/internal/dao/site"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Feedback struct {
	dto.DTOBase
	UserId  int32  `json:"user_id"`
	Content string `json:"content"`
}

func NewFeedback(feedback dao_site.Feedback) Feedback {
	return Feedback{
		DTOBase: dto.DTOBase{
			Id:            feedback.Id,
			AddedAt:       feedback.AddedAt,
			LastUpdatedAt: feedback.LastUpdatedAt,
		},
		UserId:  feedback.UserId,
		Content: feedback.Content,
	}
}
