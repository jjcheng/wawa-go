package controller

import (
	"github.com/gin-gonic/gin"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_ai_conversation "github.com/jjcheng/wawa-go/internal/feature/ai/conversation"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAIController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerRoute[[]dto_ai.Conversation, feature_ai_conversation.List](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_ai.Message, feature_ai_conversation.ListMessages](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_conversation.Delete](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai.Message, feature_ai_conversation.Chat](routerGroup, dependencies, apiGenerator)
}
