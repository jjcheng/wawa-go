package dao_account

import "github.com/jjcheng/wawa-go/internal/dao"

type Setting struct {
	dao.DAOBase
	UserId int32  `gorm:"column:user_id"`
	Name   string `gorm:"column:name"`
	Value  string `gorm:"column:value"`
}

func (Setting) TableName() string {
	return "account.settings"
}

func (setting Setting) Base() dao.DAOBase {
	return setting.DAOBase
}
