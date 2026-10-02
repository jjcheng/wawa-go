package feature_ai_conversation

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
}

func (list *List) Validate() []exception.InputException {
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai.Conversation] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_ai.Conversation](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai.Conversation](inputErrors)
	}
	conversations, err := dependencies.UnitOfWork.AIConversationRepository().ListByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai.Conversation](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := make([]dto_ai.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, dto_ai.NewConversation(conversation))
	}
	return dto.NewSuccessResponse(result)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List AI conversations",
		"List AI conversations belonging to the user.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/ai/conversations",
		true,
		true,
		types.APITagAI,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
