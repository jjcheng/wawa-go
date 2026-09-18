package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Catalog struct {
	dto.DTOBase
	BusinessAccountId int32  `json:"business_account_id"`
	MetaId            string `json:"meta_id"`
	Name              string `json:"name"`
	Vertical          string `json:"vertical"`
}

func NewCatalog(catalog dao_commerce.Catalog) Catalog {
	return Catalog{
		DTOBase: dto.DTOBase{
			Id:         catalog.Id,
			EntryDate:  catalog.EntryDate,
			LastUpdate: catalog.LastUpdate,
		},
		BusinessAccountId: catalog.BusinessAccountId,
		MetaId:            catalog.MetaId,
		Name:              catalog.Name,
		Vertical:          catalog.Vertical,
	}
}
