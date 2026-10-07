package controller

import (
	"github.com/gin-gonic/gin"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_ai_worker "github.com/jjcheng/wawa-go/internal/feature/ai_worker"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAIWorkerController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[[]dto_ai_worker.Conversation, feature_ai_worker.ListConversations](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_ai_worker.Message, feature_ai_worker.ListMessages](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_worker.DeleteConversation](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_worker.Message, feature_ai_worker.Chat](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_worker.WorkResult, feature_ai_worker.Execute](routerGroup, dependencies, apiGenerator)
}
