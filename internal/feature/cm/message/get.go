package feature_cm_message

// import (
// 	"context"

// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/helper"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// // get latest message by role in a conversation
// type Get struct {
// 	ConversationIdentifier string                `form:"conversation_identifier"`
// 	Role                   types.ChatMessageRole `form:"role"`
// }

// func (get *Get) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if get.ConversationIdentifier == "" {
// 		errors = append(errors, exception.NewInputException("conversation_identifier", "conversation_identifier is required"))
// 	}
// 	if get.Role == "" {
// 		errors = append(errors, exception.NewInputException("role", "role is required"))
// 	}
// 	return errors
// }

// func (get Get) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[*dto_cm.Message] {
// 	if errors := get.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_cm.Message](errors)
// 	}
// 	model, ex := dependencies.UnitOfWork.CMMessageRepository().GetByConversationIdentifierAndLatestRole(ctx, user.Id, get.ConversationIdentifier, get.Role)
// 	if ex != nil {
// 		return dto.NewFailedResponse[*dto_cm.Message](ex.StatusCode, ex.Message)
// 	}
// 	msg := dto_cm.NewMessage(*model)
// 	// if the content is a map, remove thinking_process
// 	if m, err := helper.DeserializeJSON[map[string]any](msg.Content); err == nil {
// 		delete(*m, "thinking_process")
// 		if json, err := helper.SerializeJSON(*m); err == nil {
// 			msg.Content = *json
// 		}
// 	}
// 	return dto.NewSuccessResponse(&msg)
// }

// func (Get) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Get latest message in a conversation by role", "Get latest message by role in a conversation such as APPOINTMENT, SHORTLIST, ALERTS, SAVED.", types.HttpRequestTypeQuery, "GET", "/cm/v1/messages/latest", true, true, types.APITagCM, nil)
// }
