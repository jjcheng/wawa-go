package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerCustomerController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto.ListResponse[dto_customer.Customer], feature_customer.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_customer.Customer, feature_customer.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[[]string, feature_customer.GetTags](routerGroup, dependencies, apiGenerator)
}
