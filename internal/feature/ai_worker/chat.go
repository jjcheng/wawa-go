package feature_ai_worker

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Chat struct {
	ConversationId int32  `json:"conversation_id" val:"required" description:"id of the conversation"`
	Message        string `json:"message" val:"required" description:"message to send"`
}

func (chat *Chat) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	chat.Message = strings.TrimSpace(chat.Message)
	if chat.Message == "" {
		inputErrors = append(inputErrors, exception.NewInputException("message", "message is required"))
	}
	// if chat.Role != types.AIMessageRoleUser && chat.Role != types.AIMessageRoleAssistant {
	// 	inputErrors = append(inputErrors, exception.NewInputException("role", "role must be USER or ASSISTANT"))
	// }
	return inputErrors
}

func (chat Chat) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_worker.Message] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := chat.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_worker.Message](inputErrors)
	}
	var conversation dao_ai_worker.Conversation
	if chat.ConversationId <= 0 {
		// create new
		newConversation := dao_ai_worker.Conversation{
			UserId: user.Id,
			Title:  chat.Message,
		}
		if err := dependencies.UnitOfWork.AIWorkerConversationRepository().Insert(ctx, &newConversation); err != nil {
			return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		conversation = newConversation
	} else {
		c, err := dependencies.UnitOfWork.AIWorkerConversationRepository().GetById(ctx, chat.ConversationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusNotFound, "conversation not found", nil)
			}
			return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		conversation = *c
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// insert this message first
	userContent, err := dto_ai_worker.SerializeWorkResultParts([]dto_ai_worker.WorkResultPart{{
		Content: strings.TrimSpace(chat.Message),
	}})
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	message := dao_ai_worker.Message{
		ConversationId: conversation.Id,
		Parts:          pq.StringArray(userContent),
		Role:           types.AIWorkerMessageRoleUser,
	}
	if err := dependencies.UnitOfWork.AIWorkerMessageRepository().Insert(ctx, &message); err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// work
	work := Work{
		ConversationId: conversation.Id,
	}
	workResponse := work.Handle(ctx, user, dependencies)
	if !workResponse.Success {
		return dto.NewFailedResponse[*dto_ai_worker.Message](workResponse.StatusCode, workResponse.Message, workResponse.Error)
	}
	// response message
	responseContents, err := dto_ai_worker.SerializeWorkResultParts(workResponse.Data.Parts)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	responseMessage := dao_ai_worker.Message{
		Feature:        workResponse.Data.Feature,
		ConversationId: conversation.Id,
		Role:           types.AIWorkerMessageRoleAssistant,
		Parts:          pq.StringArray(responseContents),
		URL:            workResponse.Data.URL,
	}
	if err := dependencies.UnitOfWork.AIWorkerMessageRepository().Insert(ctx, &responseMessage); err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_worker.NewMessage(responseMessage)
	return dto.NewSuccessResponse(&result)
}

func (chat Chat) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Chat with AI",
		"Send a message to an AI conversation.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/ai/conversations/chat",
		true,
		true,
		types.APITagAIWorker,
		nil,
		feature.NewAIWorker(false, "Hi, I am your AI worker, currently in beta! Tell me what you wish to do, I will try my best to give you the steps or provide you the form to execute the task. I may make mistakes, if I do so, please write a feedback to us, thank you!", types.AIWorkerReturnTypeText, "", ""),
	)
}
