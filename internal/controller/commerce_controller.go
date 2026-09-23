package controller

import (
	"github.com/gin-gonic/gin"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_commerce_catalog "github.com/jjcheng/wawa-go/internal/feature/commerce/catalog"
	feature_commerce_page "github.com/jjcheng/wawa-go/internal/feature/commerce/page"
	feature_commerce_website "github.com/jjcheng/wawa-go/internal/feature/commerce/website"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerCommerceController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// catalog
	registerRoute[*dto_commerce.Catalog, feature_commerce_catalog.Get](routerGroup, dependencies, apiGenerator)
	// page
	registerRoute[[]dto_commerce.Page, feature_commerce_page.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Page, feature_commerce_page.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Page, feature_commerce_page.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Page, feature_commerce_page.Update](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_commerce_page.Delete](routerGroup, dependencies, apiGenerator)
	// website
	registerRoute[[]dto_commerce.Website, feature_commerce_website.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Website, feature_commerce_website.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Website, feature_commerce_website.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Website, feature_commerce_website.GetByMetaCatalogId](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.Website, feature_commerce_website.Update](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_commerce_website.SetStatus](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_commerce_website.Sync](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_commerce_website.Delete](routerGroup, dependencies, apiGenerator)
}
