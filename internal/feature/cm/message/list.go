package feature_cm_message

// import (
// 	"context"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type List struct {
// 	ConversationIdentifier string `form:"conversation_identifier" description:"identifier of this conversation (UUID)"`
// 	ConversationId         int32  `form:"conversation_id" description:"id of the conversation, either conversation_id or conversation_identifier must present" example:"1"`
// 	Full                   bool   `description:"return all roles" example:"true"`
// }

// func (list *List) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if list.ConversationIdentifier == "" && list.ConversationId <= 0 {
// 		errors = append(errors, exception.NewInputException("conversation_id", "missing conversation_id or conversation_identifier"))
// 	}
// 	return errors
// }

// func (list List) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[[]dto_cm.Message] {
// 	if errors := list.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[[]dto_cm.Message](errors)
// 	}
// 	var messages []dao_cm.Message
// 	var ex *exception.Exception
// 	var roles []types.ChatMessageRole
// 	if !list.Full {
// 		roles = types.DisplayChatMessageRoles
// 	}
// 	if list.ConversationId > 0 {
// 		messages, ex = dependencies.UnitOfWork.CMMessageRepository().ListByUserIdAndConversationId(ctx, user.Id, list.ConversationId, roles)
// 	} else {
// 		messages, ex = dependencies.UnitOfWork.CMMessageRepository().ListByUserIdAndConversationIdentifier(ctx, user.Id, list.ConversationIdentifier, roles)
// 	}
// 	if ex != nil {
// 		return dto.NewFailedResponse[[]dto_cm.Message](ex.StatusCode, ex.Message)
// 	}
// 	items := []dto_cm.Message{}
// 	for _, item := range messages {
// 		items = append(items, dto_cm.NewMessage(item))
// 	}
// 	return dto.NewSuccessResponse(items)
// }

// func (List) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("List past messages in a conversation", "List all past messages in a conversation.", types.HttpRequestTypeQuery, "GET", "/cm/v1/messages", true, true, types.APITagCM, nil)
// }
