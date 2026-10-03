package feature_ai_conversation

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_ai_worker "github.com/jjcheng/wawa-go/internal/feature/ai/worker"
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

func (chat Chat) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.Message] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := chat.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.Message](inputErrors)
	}
	var conversation dao_ai.Conversation
	if chat.ConversationId <= 0 {
		// create new
		newConversation := dao_ai.Conversation{
			UserId: user.Id,
			Title:  chat.Message,
		}
		if err := dependencies.UnitOfWork.AIConversationRepository().Insert(ctx, &newConversation); err != nil {
			return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		conversation = newConversation
	} else {
		c, err := dependencies.UnitOfWork.AIConversationRepository().GetById(ctx, chat.ConversationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_ai.Message](http.StatusNotFound, "conversation not found", nil)
			}
			return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		conversation = *c
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// insert this message first
	userContent, err := dto_ai.SerializeWorkResultParts([]dto_ai.WorkResultPart{{
		Content: strings.TrimSpace(chat.Message),
	}})
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	message := dao_ai.Message{
		ConversationId: conversation.Id,
		Parts:          pq.StringArray(userContent),
		Role:           types.AIMessageRoleUser,
	}
	if err := dependencies.UnitOfWork.AIMessageRepository().Insert(ctx, &message); err != nil {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// work
	work := feature_ai_worker.Work{
		ConversationId: conversation.Id,
	}
	workResponse := work.Handle(ctx, user, dependencies)
	if !workResponse.Success {
		return dto.NewFailedResponse[*dto_ai.Message](workResponse.StatusCode, workResponse.Message, workResponse.Error)
	}
	// response message
	responseContents, err := dto_ai.SerializeWorkResultParts(workResponse.Data.Parts)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	responseMessage := dao_ai.Message{
		Feature:        workResponse.Data.Feature,
		ConversationId: conversation.Id,
		Role:           types.AIMessageRoleAssistant,
		Parts:          pq.StringArray(responseContents),
		URL:            workResponse.Data.URL,
	}
	if err := dependencies.UnitOfWork.AIMessageRepository().Insert(ctx, &responseMessage); err != nil {
		return dto.NewFailedResponse[*dto_ai.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai.NewMessage(responseMessage)
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
		types.APITagAI,
		nil,
		feature.NewAIWorker(false, "Hi, I am your AI worker, currently in beta! Tell me what you wish to do, I will try my best to give you the steps or provide you the form to execute the task. I may make mistakes, if I do so, please write a feedback to us, thank you!", types.AIWorkerReturnTypeText, "", ""),
	)
}
