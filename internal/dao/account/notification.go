package dao_account

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Notification struct {
	dao.DAOBase
	UserId int32                  `gorm:"column:user_id"`
	Title  string                 `gorm:"column:title"`
	Body   string                 `gorm:"column:body"`
	Type   types.NotificationType `gorm:"column:type"`
	URL    string                 `gorm:"column:url"`
	Read   bool                   `gorm:"column:read"`
}

func (Notification) TableName() string {
	return "account.notifications"
}

func (notification Notification) Base() dao.DAOBase {
	return notification.DAOBase
}
