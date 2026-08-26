package controller

import (
	"github.com/gin-gonic/gin"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_auth "github.com/jjcheng/wawa-go/internal/feature/auth"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAuthController(authGroup *gin.RouterGroup, unauthGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto_account.User, feature_auth.Login](unauthGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_auth.Me](authGroup, dependencies, apiGenerator)
	registerRoute[any, feature_auth.Logout](authGroup, dependencies, apiGenerator)
}
