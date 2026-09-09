package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_campaign "github.com/jjcheng/wawa-go/internal/feature/campaign"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerCustomerController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// customer
	registerRoute[*dto.ListResponse[dto_customer.Customer], feature_customer.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Customer, feature_customer.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Customer, feature_customer.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Customer, feature_customer.Update](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_customer.ImportResult, feature_customer.Import](routerGroup, dependencies, apiGenerator)
	registerRoute[[]string, feature_customer.GetTags](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_customer.SetStatus](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_customer.Delete](routerGroup, dependencies, apiGenerator)
	// campaigns
	registerRoute[*dto.ListResponse[dto_customer.Campaign], feature_campaign.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Campaign, feature_campaign.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Campaign, feature_campaign.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_campaign.Archive](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_campaign.Cancel](routerGroup, dependencies, apiGenerator)
}
