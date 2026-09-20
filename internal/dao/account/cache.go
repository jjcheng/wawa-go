package dao_account

import "github.com/jjcheng/wawa-go/internal/dao"

type Cache struct {
	dao.DAOBase
	Key   string `gorm:"column:key"`
	Value string `gorm:"column:value"`
}

func (Cache) TableName() string {
	return "account.cache"
}

func (cache Cache) Base() dao.DAOBase {
	return cache.DAOBase
}
