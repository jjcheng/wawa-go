package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_account_notification "github.com/jjcheng/wawa-go/internal/feature/account/notification"
	feature_account_user "github.com/jjcheng/wawa-go/internal/feature/account/user"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAccountController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*feature_account_user.Dashboard, feature_account_user.GetDashboard](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.UpdateProfile](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.ChangePassword](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.SetPassword](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_account.User, feature_account_user.Me](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_user.CloseAccount](routerGroup, dependencies, apiGenerator)
	// notification
	registerRoute[*dto.ListResponse[dto_account.Notification], feature_account_notification.List](routerGroup, dependencies, apiGenerator)
	registerRoute[int, feature_account_notification.GetUnreadCount](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_notification.Delete](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_notification.SetStatus](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AblyTokenRequest, feature_account_notification.CreateAblyToken](routerGroup, dependencies, apiGenerator)
}
