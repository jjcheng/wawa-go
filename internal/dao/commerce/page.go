package dao_commerce

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/lib/pq"
)

type Page struct {
	dao.DAOBase
	WebsiteId   int32          `gorm:"column:website_id"`
	Slug        string         `gorm:"column:slug"`
	Title       string         `gorm:"column:title"`
	Description string         `gorm:"column:description"`
	Content     string         `gorm:"column:content"`
	Nav         bool           `gorm:"column:nav"`
	Rank        int32          `gorm:"column:rank"`
	ImageUrls   pq.StringArray `gorm:"column:image_urls;type:text[]"`
}

func (Page) TableName() string {
	return "commerce.pages"
}

func (page Page) Base() dao.DAOBase {
	return page.DAOBase
}
