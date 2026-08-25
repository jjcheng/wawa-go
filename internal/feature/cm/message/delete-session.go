package feature_cm_message

// import (
// 	"context"
// 	"net/http"

// 	"github.com/jjcheng/wawa-go/internal/cfg"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type DeleteSession struct {
// 	Id int `uri:"id" val:"required" description:"id of the message" example:"1"`
// }

// func (deleteSession *DeleteSession) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if deleteSession.Id <= 0 {
// 		errors = append(errors, exception.NewInputException("id", "missing id"))
// 	}
// 	return errors
// }

// func (deleteSession DeleteSession) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[any] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := deleteSession.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[any](errors)
// 	}
// 	message, err, notFound := dependencies.UnitOfWork.CMMessageRepository().GetById(ctx, int32(deleteSession.Id))
// 	if err != nil {
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, err.Error())
// 	}
// 	if notFound {
// 		return dto.NewFailedResponse[any](http.StatusNotFound, "message not found")
// 	}
// 	ex := dependencies.UnitOfWork.CMMessageRepository().DeleteBySessionId(ctx, user.Id, message.SessionId)
// 	if ex != nil {
// 		return dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
// 	}
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

// func (DeleteSession) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Delete a session", "Delete a session and all messages", types.HttpRequestTypeUri, "DELETE", "/cm/v1/messages/session/:id", true, false, types.APITagCM, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("messages not found", http.StatusNotFound)),
// 	})
// }
