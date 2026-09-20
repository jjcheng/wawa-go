package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_broadcast "github.com/jjcheng/wawa-go/internal/feature/broadcast"
	feature_broadcast_recipient "github.com/jjcheng/wawa-go/internal/feature/broadcast_recipient"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

func registerBroadcastController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto.ListResponse[dto_customer.Broadcast], feature_broadcast.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Broadcast, feature_broadcast.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*map[types.WAMessageStatus]int, feature_broadcast.GetStatistics](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Broadcast, feature_broadcast.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_broadcast.Cancel](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_broadcast.Delete](routerGroup, dependencies, apiGenerator)
	// broadcast recipient
	registerRoute[*dto.ListResponse[dto_customer.BroadcastRecipient], feature_broadcast_recipient.List](routerGroup, dependencies, apiGenerator)
}
