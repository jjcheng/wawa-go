package dto_commerce

import (
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/lib/pq"
)

type Product struct {
	dto.DTOBase
	ProductSetId        int32          `json:"product_set_id"`
	MetaId              string         `json:"meta_id"`
	Name                string         `json:"name"`
	Price               string         `json:"price"`
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

func NewProduct(product dao_commerce.Product) Product {
	return Product{
		DTOBase: dto.DTOBase{
			Id:         product.Id,
			EntryDate:  product.EntryDate,
			LastUpdate: product.LastUpdate,
		},
		ProductSetId:        product.ProductSetId,
		MetaId:              product.MetaId,
		Name:                product.Name,
		Price:               product.Price,
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
