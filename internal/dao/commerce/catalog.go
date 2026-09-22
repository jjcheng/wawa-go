package dao_commerce

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Catalog struct {
	dao.DAOBase
	WebsiteId  int32  `gorm:"column:website_id"`
	MetaId     string `gorm:"column:meta_id"`
	Name       string `gorm:"column:name"`
	Vertical   string `gorm:"column:vertical"`
	Subscribed bool   `gorm:"column:subscribed"` // to receive webhook for product_feed (not items_batch)
	// from websites table
	WebsiteDomainName string                      `gorm:"column:website_domain_name;->"`
	WebsiteStatus     types.CommerceWebsiteStatus `gorm:"column:website_status;->"`
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
