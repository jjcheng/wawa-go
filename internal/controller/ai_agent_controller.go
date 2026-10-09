package controller

import (
	"github.com/gin-gonic/gin"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_ai_agent_faq "github.com/jjcheng/wawa-go/internal/feature/ai_agent/faq"
	feature_ai_agent_profile "github.com/jjcheng/wawa-go/internal/feature/ai_agent/profile"
	feature_ai_agent_skill "github.com/jjcheng/wawa-go/internal/feature/ai_agent/skill"
	feature_ai_agent_website "github.com/jjcheng/wawa-go/internal/feature/ai_agent/website"
	"github.com/jjcheng/wawa-go/internal/service"
)

func registerAIAgentController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// profile
	registerRoute[[]dto_ai_agent.Profile, feature_ai_agent_profile.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Profile, feature_ai_agent_profile.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Profile, feature_ai_agent_profile.Update](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Profile, feature_ai_agent_profile.UpdateBusinessInfo](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Profile, feature_ai_agent_profile.UpdateBudgets](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_profile.UpdateOtherSettings](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Profile, feature_ai_agent_profile.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_profile.Delete](routerGroup, dependencies, apiGenerator)
	// faq
	registerRoute[[]dto_ai_agent.FAQ, feature_ai_agent_faq.List](routerGroup, dependencies, apiGenerator)
	// skills
	registerRoute[[]dto_ai_agent.Skill, feature_ai_agent_skill.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Skill, feature_ai_agent_skill.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Skill, feature_ai_agent_skill.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Skill, feature_ai_agent_skill.Update](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_skill.Delete](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_skill.AddCommon](routerGroup, dependencies, apiGenerator)
	// websites
	registerRoute[[]dto_ai_agent.Website, feature_ai_agent_website.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_ai_agent.Website, feature_ai_agent_website.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_ai_agent_website.GetResult, feature_ai_agent_website.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_website.Cancel](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_ai_agent_website.Delete](routerGroup, dependencies, apiGenerator)
}
