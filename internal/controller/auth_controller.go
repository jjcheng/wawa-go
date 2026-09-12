package controller

import (
	"github.com/gin-gonic/gin"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_auth "github.com/jjcheng/wawa-go/internal/feature/auth"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAuthController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto_account.User, feature_auth.Login](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_auth.Logout](routerGroup, dependencies, apiGenerator)
}
