package feture_ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	ai_prompt "github.com/jjcheng/wawa-go/internal/feature/ai/prompt"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type RAG struct {
	Query      string
	ProfileId  int32
	Knowledges []dto_ai.Knowledge
	Skills     []dto_ai_agent.Skill
}

func (rag *RAG) Validate() []exception.InputException {
	var errors []exception.InputException
	rag.Query = strings.TrimSpace(rag.Query)
	if rag.Query == "" {
		errors = append(errors, exception.NewInputException("query", "missing query"))
	}
	if len(rag.Knowledges) == 0 {
		errors = append(errors, exception.NewInputException("knowledges", "missing knowledges"))
	}
	if rag.ProfileId <= 0 {
		errors = append(errors, exception.NewInputException("profile_id", "missing profile id"))
	}
	return errors
}

// Handle answers the query using only the given knowledges.
func (rag *RAG) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := rag.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	systemPrompt := ai_prompt.RAG
	if len(rag.Skills) > 0 {
		var skills []string
		for _, skill := range rag.Skills {
			skills = append(skills, fmt.Sprintf("- %s: %s", skill.Title, skill.Description))
		}
		systemPrompt += "\n\n## Skills you can use:\n\n" + strings.Join(skills, "\n---\n")
	}
	// get business profile
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, rag.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[string](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// gather business info
	if len(profile.BusinessInfo) > 0 {
		businessInfoList := []string{}
		for _, info := range profile.BusinessInfo {
			businessInfoList = append(businessInfoList, fmt.Sprintf("%s: %s", info["title"].(string), info["description"].(string)))
		}
		systemPrompt += "\n\n## Business info:\n\n" + strings.Join(businessInfoList, "\n\n")
	}
	items := make([]string, 0, len(rag.Knowledges))
	for _, knowledge := range rag.Knowledges {
		items = append(items, fmt.Sprintf("Q: %s\nA: %s", knowledge.Title, knowledge.Body))
	}
	systemPrompt += "\n\n## Knowledges:\n\n" + strings.Join(items, "\n---\n")
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(systemPrompt, "files/ai/rag.md")
	}
	messages := []service.DashScopeMessage{
		{Role: service.DashScopeChatMessageRoleSystem, Content: systemPrompt},
		{Role: service.DashScopeChatMessageRoleUser, Content: rag.Query},
	}
	response, err := dependencies.LLM.Chat(ctx, service.DashScopeChatModelQwen38Flash, messages, 0.2, nil, 0, service.DashScopeResponseFormatText)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(response.Output.Choices) == 0 {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("llm returned no choices"))
	}
	text := replyText(response.Output.Choices[0].Message.Content)
	if text == "" {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("llm returned an empty answer"))
	}
	return dto.NewSuccessResponse(text)
}
