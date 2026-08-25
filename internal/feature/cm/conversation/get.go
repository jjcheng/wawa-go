package feature_cm_conversation

// import (
// 	"context"
// 	"net/http"

// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type Get struct {
// 	Id int32 `uri:"id" description:"id of this conversation" val:"required" example:"1"`
// }

// func (get *Get) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if get.Id <= 0 {
// 		errors = append(errors, exception.NewInputException("id", "missing id"))
// 	}
// 	return errors
// }

// func (get Get) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[*dto_cm.Conversation] {
// 	if errors := get.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_cm.Conversation](errors)
// 	}
// 	// get conversation
// 	conversation, ex := dependencies.UnitOfWork.CMConversationRepository().GetByUserIdAndId(ctx, user.Id, get.Id)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[*dto_cm.Conversation](ex.StatusCode, "conversation not found")
// 		}
// 		return dto.NewFailedResponse[*dto_cm.Conversation](ex.StatusCode, ex.Message)
// 	}
// 	d := dto_cm.NewConversation(*conversation)
// 	return dto.NewSuccessResponse(&d)
// }

// func (Get) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Get conversation details", "Get details of a conversation.", types.HttpRequestTypeUri, "GET", "/cm/v1/conversations/:id", true, true, types.APITagCM, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("conversation not found", http.StatusNotFound)),
// 	})
// }
