package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Set struct {
	dto.DTOBase
	CatalogId int32  `json:"catalog_id"`
	MetaId    string `json:"meta_id"`
	Name      string `json:"name"`
}

func NewProductSet(set dao_commerce.Set) Set {
	return Set{
		DTOBase: dto.DTOBase{
			Id:         set.Id,
			EntryDate:  set.EntryDate,
			LastUpdate: set.LastUpdate,
		},
		CatalogId: set.CatalogId,
		MetaId:    set.MetaId,
		Name:      set.Name,
	}
}
