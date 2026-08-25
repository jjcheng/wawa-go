package feature_cm_conversation

// import (
// 	"context"
// 	"net/http"

// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type Delete struct {
// 	Get
// }

// func (delete Delete) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[any] {
// 	if errors := delete.Get.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[any](errors)
// 	}
// 	getResponse := delete.Get.Handle(ctx, user, dependencies)
// 	if !getResponse.Success {
// 		return dto.NewFailedResponse[any](getResponse.StatusCode, getResponse.Message)
// 	}
// 	err := dependencies.UnitOfWork.CMConversationRepository().DeleteById(ctx, getResponse.Data.Id)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err)
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, "error deleting conversation")
// 	}
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

// func (Delete) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Delete a conversation", "Delete a conversation and any associated messages.", types.HttpRequestTypeUriQuery, "DELETE", "/cm/v1/conversations/:id", true, true, types.APITagCM, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("conversation not found", http.StatusNotFound)),
// 	})
// }
