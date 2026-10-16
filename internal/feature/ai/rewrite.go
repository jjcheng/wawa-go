package feture_ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	ai_prompt "github.com/jjcheng/wawa-go/internal/feature/ai/prompt"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Rewrite struct {
	Messages       []dto_wa.Message
	ThinkingEffort *service.DashScopeQwenThiningEffort
}

func (rewrite *Rewrite) Validate() []exception.InputException {
	var errors []exception.InputException
	if len(rewrite.Messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
	}
	return errors
}

// return a list of decomposed tasks
func (rewrite *Rewrite) Handle(ctx context.Context, dependencies *service.Dependencies) dto.Response[[]string] {
	if errors := rewrite.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]string](errors)
	}
	if len(rewrite.Messages) == 1 {
		return dto.NewSuccessResponse([]string{rewrite.Messages[0].Text()})
	}
	messages := []service.DashScopeMessage{}
	messages = append(messages, service.DashScopeMessage{
		Role:    service.DashScopeChatMessageRoleSystem,
		Content: ai_prompt.Rewrite,
	})
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(ai_prompt.Rewrite, "files/ai/rewrite.md")
	}
	for _, message := range rewrite.Messages {
		text := message.Text()
		if text == "" {
			continue
		}
		role := service.DashScopeChatMessageRoleUser
		if message.Sending {
			role = service.DashScopeChatMessageRoleAssistant
		}
		messages = append(messages, service.DashScopeMessage{Role: role, Content: text})
	}
	// take only the last 30 messages
	if len(messages) > 30 {
		messages = messages[len(messages)-30:]
	}
	response, err := dependencies.LLM.Chat(ctx, service.DashScopeChatModelQwen38Flash, messages, 0, rewrite.ThinkingEffort, 0, service.DashScopeResponseFormatJSONObject)
	if err != nil {
		return dto.NewFailedResponse[[]string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(response.Output.Choices) == 0 {
		return dto.NewFailedResponse[[]string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("llm returned no choices"))
	}
	var result struct {
		Tasks []string `json:"tasks"`
	}
	content := replyText(response.Output.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return dto.NewFailedResponse[[]string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("error parsing rewrite result %q: %w", content, err))
	}
	tasks := make([]string, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		if task = strings.TrimSpace(task); task != "" {
			tasks = append(tasks, task)
		}
	}
	tasks = helper.Distinct(tasks)
	return dto.NewSuccessResponse(tasks)
}
