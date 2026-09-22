package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/lib/pq"
)

type GenericProduct struct {
	dto.DTOBase
	SetId               int32          `json:"set_id"`
	MetaId              string         `json:"meta_id"`
	Name                string         `json:"name"`
	Price               string         `json:"price"`
	SalePrice           string         `json:"sale_price"`
	Availability        string         `json:"availability"`
	Condition           string         `json:"condition"`
	Currency            string         `json:"currency"`
	Description         string         `json:"description"`
	FBCategory          string         `json:"fb_category"`
	Gender              string         `json:"gender"`
	ImageUrl            string         `json:"image_url"`
	RetailerId          string         `json:"retailer_id"`
	Url                 string         `json:"url"`
	AdditionalImageUrls pq.StringArray `json:"additional_image_urls"`
}

func NewProduct(product dao_commerce.GenericProduct) GenericProduct {
	return GenericProduct{
		DTOBase: dto.DTOBase{
			Id:            product.Id,
			AddedAt:       product.AddedAt,
			LastUpdatedAt: product.LastUpdatedAt,
		},
		SetId:               product.SetId,
		MetaId:              product.MetaId,
		Name:                product.Name,
		Price:               product.Price,
		SalePrice:           product.SalePrice,
		Availability:        product.Availability,
		Condition:           product.Condition,
		Currency:            product.Currency,
		Description:         product.Description,
		FBCategory:          product.FBCategory,
		Gender:              product.Gender,
		ImageUrl:            product.ImageUrl,
		RetailerId:          product.RetailerId,
		Url:                 product.Url,
		AdditionalImageUrls: product.AdditionalImageUrls,
	}
}
