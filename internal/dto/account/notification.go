package dto_account

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Notification struct {
	dto.DTOBase
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Type  types.NotificationType `json:"type"`
	URL   string                 `json:"url"`
	Read  bool                   `json:"read"`
}

func NewNotification(notification dao_account.Notification) Notification {
	return Notification{
		DTOBase: dto.DTOBase{
			Id:         notification.Id,
			EntryDate:  notification.EntryDate,
			LastUpdate: notification.LastUpdate,
		},
		Title: notification.Title,
		Body:  notification.Body,
		Type:  notification.Type,
		URL:   notification.URL,
		Read:  notification.Read,
	}
}
