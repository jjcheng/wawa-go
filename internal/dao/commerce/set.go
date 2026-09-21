package dao_commerce

import "github.com/jjcheng/wawa-go/internal/dao"

type Set struct {
	dao.DAOBase
	CatalogId int32  `gorm:"column:catalog_id"`
	MetaId    string `gorm:"column:meta_id"`
	Name      string `gorm:"column:name"`
	Rank      int32  `gorm:"column:rank"`
}

func (Set) TableName() string {
	return "commerce.sets"
}

func (set Set) Base() dao.DAOBase {
	return set.DAOBase
}
