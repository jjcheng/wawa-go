package controller

import (
	dto_site "github.com/jjcheng/wawa-go/internal/dto/site"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_site_feedback "github.com/jjcheng/wawa-go/internal/feature/site/feedback"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registerSiteController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[*dto_site.Feedback, feature_site_feedback.Create](routerGroup, dependencies, apiGenerator)
}
