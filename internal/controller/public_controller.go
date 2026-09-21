package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_public "github.com/jjcheng/wawa-go/internal/feature/public"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerPublicController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto_commerce.Website, feature_public.Ping](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_commerce.GenericProduct, feature_public.GetProduct](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_commerce.Set, feature_public.ListSets](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[dto_commerce.GenericProduct], feature_public.ListProducts](routerGroup, dependencies, apiGenerator)
	registerRoute[string, feature_public.GetWebsiteWALink](routerGroup, dependencies, apiGenerator)
}
