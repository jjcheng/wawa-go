package dao_account

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
)

type Cache struct {
	dao.DAOBase
	Key       string    `gorm:"column:key"`
	Value     string    `gorm:"column:value"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
}

func (Cache) TableName() string {
	return "account.cache"
}

func (cache Cache) Base() dao.DAOBase {
	return cache.DAOBase
}
