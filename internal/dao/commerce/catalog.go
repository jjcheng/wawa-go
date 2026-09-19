package dao_commerce

import "github.com/jjcheng/wawa-go/internal/dao"

type Catalog struct {
	dao.DAOBase
	BusinessAccountId int32  `gorm:"column:business_account_id"`
	MetaId            string `gorm:"column:meta_id"`
	Name              string `gorm:"column:name"`
	Vertical          string `gorm:"column:vertical"`
}

// adoptable_pets
// apps_and_software
// articles_and_publications
// commerce
// destinations
// flights
// generic
// home_listings
// hotels
// local_service_businesses
// media_titles
// offer_items
// services
// offline_commerce
// transactable_items
// vehicles

func (Catalog) TableName() string {
	return "commerce.catalogs"
}

func (catalog Catalog) Base() dao.DAOBase {
	return catalog.DAOBase
}
