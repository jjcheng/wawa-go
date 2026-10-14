package feture_ai

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Chat struct {
	AgentProfile *dto_ai_agent.Profile
	Messages     []dto_wa.Message
}

func (chat *Chat) Validate() []exception.InputException {
	var errors []exception.InputException
	if chat.AgentProfile == nil {
		errors = append(errors, exception.NewInputException("agent_profile", "missing agent profile"))
	}
	if len(chat.Messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
	}
	return errors
}

func (chat Chat) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Message] {
	if errors := chat.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Message](errors)
	}
}
