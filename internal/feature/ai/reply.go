package feture_ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	ai_prompt "github.com/jjcheng/wawa-go/internal/feature/ai/prompt"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Reply struct {
	Messages       []dto_wa.Message
	Profile        *dto_ai_agent.Profile
	Skills         []dto_ai_agent.Skill
	Temperature    *float32
	ThikingEffort  *service.DashScopeQwenThiningEffort
	AdditionalData string
	Instruction    string // used to override any existing instruction in system prompt
}

func (reply *Reply) Validate() []exception.InputException {
	var errors []exception.InputException
	if len(reply.Messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
	}
	if reply.Profile == nil {
		errors = append(errors, exception.NewInputException("profile", "missing profile"))
	}
	reply.AdditionalData = strings.TrimSpace(reply.AdditionalData)
	reply.Instruction = strings.TrimSpace(reply.Instruction)
	if reply.Temperature == nil {
		reply.Temperature = helper.ConvertToPointer(float32(0.5))
	}
	return errors
}

func (reply *Reply) Handle(ctx context.Context, dependencies *service.Dependencies) dto.Response[string] {
	if errors := reply.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	systemPrompt := ai_prompt.Reply
	if len(reply.Skills) > 0 {
		var skills []string
		for _, skill := range reply.Skills {
			skills = append(skills, fmt.Sprintf("%s", skill.Description))
		}
		systemPrompt += "\n\n## Skills you can use:\n\n" + strings.Join(skills, "\n---\n")
	}
	// gather business info
	if len(reply.Profile.BusinessInfo) > 0 {
		businessInfoList := []string{}
		for _, info := range reply.Profile.BusinessInfo {
			businessInfoList = append(businessInfoList, fmt.Sprintf("%s: %s", info.Title, info.Description))
		}
		systemPrompt += "\n\n## Business info:\n\n" + strings.Join(businessInfoList, "\n\n")
	}
	// append additional data
	if reply.AdditionalData != "" {
		systemPrompt += "\n\n## Additional data:\n\n" + reply.AdditionalData
	}
	// append instruction
	if reply.Instruction != "" {
		systemPrompt += "\n\n## Instruction (should override any piror statement):\n\n" + reply.Instruction
	}
	// save to file
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(systemPrompt, "files/ai/reply.md")
	}
	// append messages
	messages := []service.DashScopeMessage{}
	messages = append(messages, service.DashScopeMessage{
		Role:    service.DashScopeChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, message := range reply.Messages {
		role := service.DashScopeChatMessageRoleUser
		if message.Sending {
			role = service.DashScopeChatMessageRoleAssistant
		}
		messages = append(messages, service.DashScopeMessage{Role: role, Content: message.Text()})
	}
	// take only the last 30 messages
	if len(messages) > 30 {
		messages = messages[len(messages)-30:]
	}
	response, err := dependencies.LLM.Chat(ctx, service.DashScopeChatModelQwen38Flash, messages, *reply.Temperature, reply.ThikingEffort, 0, service.DashScopeResponseFormatText)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(response.Output.Choices) == 0 {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("llm returned no choices"))
	}
	text := replyText(response.Output.Choices[0].Message.Content)
	if text == "" {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, fmt.Errorf("llm returned an empty reply"))
	}
	return dto.NewSuccessResponse(text)
}

// replyText handles both a plain string and the multimodal [{"text": "..."}] content shape.
func replyText(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		var parts []string
		for _, item := range value {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	}
	return ""
}
