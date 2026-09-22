package dto_customer

import (
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Broadcast struct {
	dto.DTOBase
	Name                string                `json:"name"`
	SendDate            time.Time             `json:"send_date"`
	WATemplateId        string                `json:"wa_template_id"`
	UserId              int32                 `json:"user_id"`
	RecipientCount      int32                 `json:"recipient_count"`
	Status              types.BroadcastStatus `json:"status"`
	Token               string                `json:"token"`
	SendTemplatePayload map[string]any        `json:"send_template_payload"`
	TemplatePayload     map[string]any        `json:"template_payload"`
}

func NewBroadcast(broadcast dao_customer.Broadcast) Broadcast {
	c := Broadcast{
		DTOBase: dto.DTOBase{
			Id:            broadcast.Id,
			AddedAt:       broadcast.AddedAt,
			LastUpdatedAt: broadcast.LastUpdatedAt,
		},
		Name:                broadcast.Name,
		WATemplateId:        broadcast.WATemplateId,
		UserId:              broadcast.UserId,
		RecipientCount:      broadcast.RecipientCount,
		Status:              broadcast.Status,
		Token:               broadcast.Token,
		SendTemplatePayload: broadcast.SendTemplatePayload,
		SendDate:            broadcast.SendDate,
		TemplatePayload:     broadcast.TemplatePayload,
	}
	return c
}
