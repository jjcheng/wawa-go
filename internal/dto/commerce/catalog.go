package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Catalog struct {
	dto.DTOBase
	WebsiteId int32  `json:"website_id"`
	MetaId    string `json:"meta_id"`
	Name      string `json:"name"`
	Vertical  string `json:"vertical"`
}

func NewCatalog(catalog dao_commerce.Catalog) Catalog {
	return Catalog{
		DTOBase: dto.DTOBase{
			Id:            catalog.Id,
			AddedAt:       catalog.AddedAt,
			LastUpdatedAt: catalog.LastUpdatedAt,
		},
		WebsiteId: catalog.WebsiteId,
		MetaId:    catalog.MetaId,
		Name:      catalog.Name,
		Vertical:  catalog.Vertical,
	}
}
