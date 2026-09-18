package dao_commerce

import "github.com/jjcheng/wawa-go/internal/dao"

type ProductSet struct {
	dao.DAOBase
	CatalogId int32  `gorm:"column:catalog_id"`
	MetaId    string `gorm:"column:meta_id"`
	Name      string `gorm:"column:name"`
}

func (ProductSet) TableName() string {
	return "commerce.product_sets"
}

func (productSet ProductSet) Base() dao.DAOBase {
	return productSet.DAOBase
}
