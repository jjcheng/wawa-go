package dto_commerce

import (
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Website struct {
	dto.DTOBase
	BusinessAccountId int32                       `json:"business_account_id"`
	DomainName        string                      `json:"domain_name"`
	Status            types.CommerceWebsiteStatus `json:"status"`
	URL               string                      `json:"url"`
	// from phone number business profile
	About             string `json:"about"`
	Description       string `json:"description"`
	ProfilePictureURL string `json:"profile_picture_url"`
	Address           string `json:"address"`
	Email             string `json:"email"`
	Vertical          string `json:"vertical"`
	// from catalogs table
	MetaCatalogId string `json:"meta_catalog_id"`
	CatalogName   string `json:"catalog_name"`
}

func NewWebsite(website dao_commerce.Website) Website {
	w := Website{
		DTOBase: dto.DTOBase{
			Id:         website.Id,
			EntryDate:  website.EntryDate,
			LastUpdate: website.LastUpdate,
		},
		Status:            website.Status,
		MetaCatalogId:     website.MetaCatalogId,
		BusinessAccountId: website.BusinessAccountId,
		DomainName:        website.DomainName,
		CatalogName:       website.CatalogName,
		About:             website.About,
		Description:       website.Description,
		ProfilePictureURL: website.ProfilePictureURL,
		Address:           website.Address,
		Email:             website.Email,
		Vertical:          website.Vertical,
	}
	w.URL = w.FullUrl()
	return w
}

func (website *Website) FullUrl() string {
	if strings.HasPrefix(website.DomainName, "https://") {
		return website.DomainName
	}
	return fmt.Sprintf("https://%s.%s", website.DomainName, cfg.Default().Commerce.WebsiteDomain)
}
