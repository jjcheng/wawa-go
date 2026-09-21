package dao_commerce

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/lib/pq"
)

type GenericProduct struct {
	dao.DAOBase
	SetId               int32          `gorm:"column:set_id"`
	MetaId              string         `gorm:"column:meta_id"`
	Name                string         `gorm:"column:name"`
	Price               string         `gorm:"column:price"`
	Availability        string         `gorm:"column:availability"`
	Condition           string         `gorm:"column:condition"`
	Currency            string         `gorm:"column:currency"`
	Description         string         `gorm:"column:description"`
	FBCategory          string         `gorm:"column:fb_category"`
	Gender              string         `gorm:"column:gender"`
	ImageUrl            string         `gorm:"column:image_url"`
	RetailerId          string         `gorm:"column:retailer_id"`
	Url                 string         `gorm:"column:url"`
	AdditionalImageUrls pq.StringArray `gorm:"column:additional_image_urls;type:text[]"`
}

func (GenericProduct) TableName() string {
	return "commerce.generic_products"
}

func (product GenericProduct) Base() dao.DAOBase {
	return product.DAOBase
}
