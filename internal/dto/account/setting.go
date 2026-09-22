package dto_account

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Setting struct {
	dto.DTOBase
	UserId int32  `json:"user_id" val:"required" description:"id of the user"`
	Name   string `json:"name" val:"required" example:"ASK_SYSTEM_MESSAGE"`
	Value  string `json:"value" val:"required" example:"Your role is ...."`
}

func NewSetting(setting dao_account.Setting) Setting {
	return Setting{
		DTOBase: dto.DTOBase{
			Id:            setting.Id,
			AddedAt:       setting.AddedAt,
			LastUpdatedAt: setting.LastUpdatedAt,
		},
		Name:  setting.Name,
		Value: setting.Value,
	}
}
