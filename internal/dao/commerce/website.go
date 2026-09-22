package dao_commerce

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

// each website belongs to the entire business portfolio not an individual user
type Website struct {
	dao.DAOBase
	BusinessAccountId   int32                       `gorm:"column:business_account_id"`
	DomainName          string                      `gorm:"column:domain_name"`
	Status              types.CommerceWebsiteStatus `gorm:"column:status"`
	ProductsLastSynedAt time.Time                   `gorm:"column:products_last_synced_at"`
	// phone number profile
	About             string `gorm:"column:about"`
	Description       string `gorm:"column:description"`
	ProfilePictureURL string `gorm:"column:profile_picture_url"`
	Address           string `gorm:"column:address"`
	Email             string `gorm:"column:email"`
	Vertical          string `gorm:"column:vertical"`
	ContactText       string `gorm:"column:contact_text"`
	// from catalogs table
	MetaCatalogId string `gorm:"column:meta_catalog_id;->"`
	CatalogName   string `gorm:"column:catalog_name;->"`
}

func (Website) TableName() string {
	return "commerce.websites"
}

func (website Website) Base() dao.DAOBase {
	return website.DAOBase
}
