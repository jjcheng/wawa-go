package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Page struct {
	dto.DTOBase
	WebsiteId   int32    `json:"website_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Slug        string   `json:"slug"`
	Content     string   `json:"content"`
	Nav         bool     `json:"nav"`
	Rank        int32    `json:"rank"`
	ImageUrls   []string `json:"image_urls"`
}

func NewPage(page dao_commerce.Page) Page {
	return Page{
		DTOBase: dto.DTOBase{
			Id:            page.Id,
			AddedAt:       page.AddedAt,
			LastUpdatedAt: page.LastUpdatedAt,
		},
		WebsiteId:   page.WebsiteId,
		Title:       page.Title,
		Description: page.Description,
		Slug:        page.Slug,
		Content:     page.Content,
		Nav:         page.Nav,
		Rank:        page.Rank,
		ImageUrls:   page.ImageUrls,
	}
}

type NavBarItem struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}
