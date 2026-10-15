package feture_ai

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
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

func (rewrite *Rewrite) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := rewrite.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}

	// TODO: implement
	return dto.NewSuccessResponse("")
}
