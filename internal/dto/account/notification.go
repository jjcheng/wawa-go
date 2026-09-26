package dto_account

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Notification struct {
	dto.DTOBase
	Type     types.NotificationType     `json:"type"`
	Category types.NotificationCategory `json:"category"`
	IconType types.NotificationIconType `json:"icon_type"`
	Title    string                     `json:"title"`
	Body     string                     `json:"body"`
	URL      string                     `json:"url"`
	Read     bool                       `json:"read"`
}

func NewNotification(notification dao_account.Notification) Notification {
	return Notification{
		DTOBase: dto.DTOBase{
			Id:            notification.Id,
			AddedAt:       notification.AddedAt,
			LastUpdatedAt: notification.LastUpdatedAt,
		},
		IconType: notification.IconType,
		Category: notification.Category,
		Title:    notification.Title,
		Body:     notification.Body,
		Type:     notification.Type,
		URL:      notification.URL,
		Read:     notification.Read,
	}
}
