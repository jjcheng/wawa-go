package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type ProductSet struct {
	dto.DTOBase
	CatalogId int32  `json:"catalog_id"`
	MetaId    string `json:"meta_id"`
	Name      string `json:"name"`
}

func NewProductSet(productSet dao_commerce.ProductSet) ProductSet {
	return ProductSet{
		DTOBase: dto.DTOBase{
			Id:         productSet.Id,
			EntryDate:  productSet.EntryDate,
			LastUpdate: productSet.LastUpdate,
		},
		CatalogId: productSet.CatalogId,
		MetaId:    productSet.MetaId,
		Name:      productSet.Name,
	}
}
