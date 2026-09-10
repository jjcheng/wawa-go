package controller

import (
	"github.com/gin-gonic/gin"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_account_admin "github.com/jjcheng/wawa-go/internal/feature/account/admin"
	feature_account_user "github.com/jjcheng/wawa-go/internal/feature/account/user"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAccountController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*feature_account_user.Dashboard, feature_account_user.GetDashboard](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.UpdateProfile](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_admin.UpdateStatus](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.ChangePassword](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.SetPassword](routerGroup, dependencies, apiGenerator)
}
