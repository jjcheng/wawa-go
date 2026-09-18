package dao_commerce

import "github.com/jjcheng/wawa-go/internal/dao"

type Catalog struct {
	dao.DAOBase
	BusinessAccountId int32  `gorm:"column:business_account_id"`
	MetaId            string `gorm:"column:meta_id"`
	Name              string `gorm:"column:name"`
	Vertical          string `gorm:"column:vertical"`
}

func (Catalog) TableName() string {
	return "commerce.catalogs"
}

func (catalog Catalog) Base() dao.DAOBase {
	return catalog.DAOBase
}
