package dao_commerce

import "github.com/jjcheng/wawa-go/internal/dao"

// each website belongs to the entire business portfolio not an individual user
type Website struct {
	dao.DAOBase
	CatalogId  int32  `gorm:"column:catalog_id"`
	ThemeId    int32  `gorm:"column:theme_id"`
	DomainName string `gorm:"column:domain_name"`
}

func (Website) TableName() string {
	return "commerce.websites"
}

func (website Website) Base() dao.DAOBase {
	return website.DAOBase
}
