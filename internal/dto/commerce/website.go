package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Website struct {
	dto.DTOBase
	CatalogId  int32  `json:"catalog_id"`
	ThemeId    int32  `json:"theme_id"`
	DomainName string `json:"domain_name"`
}

func NewWebsite(website dao_commerce.Website) Website {
	return Website{
		DTOBase: dto.DTOBase{
			Id:         website.Id,
			EntryDate:  website.EntryDate,
			LastUpdate: website.LastUpdate,
		},
		CatalogId:  website.CatalogId,
		ThemeId:    website.ThemeId,
		DomainName: website.DomainName,
	}
}
