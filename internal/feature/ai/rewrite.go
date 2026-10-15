package feture_ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	ai_prompt "github.com/jjcheng/wawa-go/internal/feature/ai/prompt"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Rewrite struct {
	Messages []dto_wa.Message
}

func (rewrite *Rewrite) Validate() []exception.InputException {
	var errors []exception.InputException
	if len(rewrite.Messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
	}
	return errors
}

// Handle rewrites the latest message into standalone tasks. An empty result means there is nothing to handle.
func (rewrite *Rewrite) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[[]string] {
	if errors := rewrite.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]string](errors)
	}
	messages := make([]service.DashScopeMessage, 0, len(rewrite.Messages)+1)
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
	if len(messages) == 1 {
		return dto.NewSuccessResponse([]string{})
	}
	response, err := dependencies.LLM.Chat(ctx, service.DashScopeChatModelQwen38Flash, messages, 0, nil, 0, service.DashScopeResponseFormatJSONObject)
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
	return dto.NewSuccessResponse(tasks)
}
