package feature_ai_worker

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListConversations struct {
}

func (listConversations *ListConversations) Validate() []exception.InputException {
	return nil
}

func (listConversations ListConversations) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai_worker.Conversation] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_ai_worker.Conversation](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := listConversations.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai_worker.Conversation](inputErrors)
	}
	conversations, err := dependencies.UnitOfWork.AIWorkerConversationRepository().ListByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai_worker.Conversation](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := make([]dto_ai_worker.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, dto_ai_worker.NewConversation(conversation))
	}
	return dto.NewSuccessResponse(result)
}

func (ListConversations) APISettings() feature.APISettings {
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
