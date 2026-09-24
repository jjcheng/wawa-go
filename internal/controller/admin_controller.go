package controller

import (
	"github.com/gin-gonic/gin"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_account_admin "github.com/jjcheng/wawa-go/internal/feature/account/admin"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAdminController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[[]dto_account.User, feature_account_admin.ListUsers](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.PhoneNumber, feature_account_admin.ListUnassignedPhoneNumbers](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_admin.AssignPhoneNumbers](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_admin.SetUserStatus](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_account_admin.SetUserType](routerGroup, dependencies, apiGenerator)
}
