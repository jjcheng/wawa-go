package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_campaign "github.com/jjcheng/wawa-go/internal/feature/campaign"
	feature_campaign_recipient "github.com/jjcheng/wawa-go/internal/feature/campaign_recipient"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerCampaignController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto.ListResponse[dto_customer.Campaign], feature_campaign.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Campaign, feature_campaign.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Campaign, feature_campaign.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_campaign.Cancel](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_campaign.Delete](routerGroup, dependencies, apiGenerator)
	// campaign recipient
	registerRoute[*dto.ListResponse[dto_customer.CampaignRecipient], feature_campaign_recipient.List](routerGroup, dependencies, apiGenerator)
}
