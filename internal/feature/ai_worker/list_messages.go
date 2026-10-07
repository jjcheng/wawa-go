package feature_ai_worker

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type ListMessages struct {
	ConversationId int32 `uri:"conversation_id" val:"required" description:"id of the conversation"`
}

func (listMessages *ListMessages) Validate() []exception.InputException {
	if listMessages.ConversationId <= 0 {
		return []exception.InputException{exception.NewInputException("conversation_id", "invalid conversation id")}
	}
	return nil
}

func (listMessages ListMessages) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai_worker.Message] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_ai_worker.Message](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := listMessages.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_ai_worker.Message](inputErrors)
	}
	conversation, err := dependencies.UnitOfWork.AIWorkerConversationRepository().GetById(ctx, listMessages.ConversationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_ai_worker.Message](http.StatusNotFound, "conversation not found", nil)
		}
		return dto.NewFailedResponse[[]dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[[]dto_ai_worker.Message](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	messages, err := dependencies.UnitOfWork.AIWorkerMessageRepository().ListByConversationId(ctx, listMessages.ConversationId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := make([]dto_ai_worker.Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, dto_ai_worker.NewMessage(message))
	}
	return dto.NewSuccessResponse(result)
}

func (ListMessages) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List AI conversation messages",
		"List messages belonging to an AI conversation owned by the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/ai/conversations/:conversation_id/messages",
		true,
		true,
		types.APITagAI,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("conversation not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
