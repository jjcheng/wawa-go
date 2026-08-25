package feature_cm_conversation

// import (
// 	"context"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	dto_core "github.com/jjcheng/wawa-go/internal/dto/core"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/helper"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type List struct {
// }

// func (list *List) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	return errors
// }

// func (list List) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[[]dto_cm.Conversation] {
// 	if errors := list.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[[]dto_cm.Conversation](errors)
// 	}
// 	conversationDAOs, ex := dependencies.UnitOfWork.CMConversationRepository().ListByUserId(ctx, user.Id)
// 	if ex != nil {
// 		return dto.NewFailedResponse[[]dto_cm.Conversation](ex.StatusCode, ex.Message)
// 	}
// 	if len(conversationDAOs) == 0 {
// 		return dto.NewSuccessResponse([]dto_cm.Conversation{})
// 	}
// 	// get last messages
// 	conversationIds := helper.Map(conversationDAOs, func(c dao_cm.Conversation) int32 {
// 		return c.Id
// 	})
// 	messages, ex := dependencies.UnitOfWork.CMMessageRepository().ListLastMessagesByConversationIds(ctx, conversationIds, types.DisplayChatMessageRoles)
// 	if ex != nil {
// 		return dto.NewFailedResponse[[]dto_cm.Conversation](ex.StatusCode, ex.Message)
// 	}
// 	conversations := make([]dto_cm.Conversation, len(conversationDAOs))
// 	for i, entity := range conversationDAOs {
// 		conv := dto_cm.NewConversation(entity)
// 		conversations[i] = conv
// 		// get last messages
// 		if lastMessage := helper.First(messages, func(cm dao_cm.Message) bool {
// 			return cm.ConversationId == conversations[i].Id
// 		}); lastMessage != nil {
// 			lastMessage := dto_core.NewChatMessageFromDB(lastMessage.Identifier, lastMessage.Role, lastMessage.Content, lastMessage.DateTime, nil)
// 			conversations[i].LastMessage = &lastMessage
// 		}
// 	}
// 	return dto.NewSuccessResponse(conversations)
// }

// func (List) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("List all conversations", "List all conversations by the user.", types.HttpRequestTypeQuery, "GET", "/cm/v1/conversations", true, true, types.APITagCM, nil)
// }
