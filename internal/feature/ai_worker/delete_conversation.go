package feature_ai_worker

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type DeleteConversation struct {
	Id int32 `uri:"id" val:"required" description:"id of the conversation"`
}

func (deleteConversation *DeleteConversation) Validate() []exception.InputException {
	if deleteConversation.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid conversation id")}
	}
	return nil
}

func (deleteConversation DeleteConversation) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := deleteConversation.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	conversation, err := dependencies.UnitOfWork.AIWorkerConversationRepository().GetById(ctx, deleteConversation.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "conversation not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.UnitOfWork.AIWorkerConversationRepository().DeleteById(ctx, conversation.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (DeleteConversation) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete an AI conversation",
		"Delete an AI conversation belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/ai/conversations/:id",
		true,
		true,
		types.APITagAI,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("conversation not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
