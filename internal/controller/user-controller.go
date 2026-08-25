package controller

import (
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registerUserController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// registerRoute[[]dto_ai.User, feature_user.List](routerGroup, dependencies, apiGenerator)
	// registerRoute[*dto_ai.User, feature_user.Create](routerGroup, dependencies, apiGenerator)
	// registerRoute[any, feature_user.UpdateDescription](routerGroup, dependencies, apiGenerator)
	// registerRoute[any, feature_user.UpdateStatus](routerGroup, dependencies, apiGenerator)
	// registerRoute[*dto_ai.User, feature_user.RotateKeys](routerGroup, dependencies, apiGenerator)
	// registerRoute[any, feature_user.Delete](routerGroup, dependencies, apiGenerator)
	// registerRoute[*dto_ai.User, feature_user.Profile](routerGroup, dependencies, apiGenerator)
}
