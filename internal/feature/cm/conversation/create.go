package feature_cm_conversation

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"strings"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/feature"
// 	"github.com/jjcheng/wawa-go/internal/helper"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"

// 	"github.com/google/uuid"
// )

// type Create struct {
// 	Identifier string                   `json:"identifier" val:"required" description:"uuid of this conversation" example:""`
// 	Channel    types.ChatMessageChannel `json:"channel" val:"required" description:"WHATSAPP, WEB" example:"WHATSAPP"`
// 	Source     string                   `json:"source" val:"required" description:"domain name or campaign id" example:"https://www.pomenandpeak.com/qwrewokjfdsi"`
// }

// func (create *Create) Validate() []exception.InputException {
// 	create.Source = strings.TrimSpace(create.Source)
// 	create.Identifier = strings.TrimSpace(create.Identifier)
// 	errors := []exception.InputException{}
// 	if err := uuid.Validate(create.Identifier); err != nil {
// 		errors = append(errors, exception.NewInputException("identifier", fmt.Sprintf("identifier error: %s", err.Error())))
// 	}
// 	if create.Channel == "" {
// 		errors = append(errors, exception.NewInputException("channel", "missing channel"))
// 	} else {
// 		if !helper.Any(types.ChatMessageChannels, func(r types.ChatMessageChannel) bool {
// 			return r == create.Channel
// 		}) {
// 			valid := strings.Join(helper.Map(types.ChatMessageChannels, func(r types.ChatMessageChannel) string {
// 				return string(r)
// 			}), ", ")
// 			errors = append(errors, exception.NewInputException("channel", fmt.Sprintf("invalid channel, valid types are: %s", valid)))
// 		}
// 	}
// 	if create.Source == "" {
// 		errors = append(errors, exception.NewInputException("source", "missing source"))
// 	}
// 	return errors
// }

// func (create Create) Handle(ctx context.Context, u *dto_ai.User, dependencies *service.Dependencies) dto.Response[*dto_cm.Conversation] {
// 	if errors := create.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_cm.Conversation](errors)
// 	}
// 	// check conversation exists
// 	exist, ex := dependencies.UnitOfWork.CMConversationRepository().CheckConversationExist(ctx, u.Id, create.Identifier)
// 	if ex != nil {
// 		return dto.NewFailedResponse[*dto_cm.Conversation](ex.StatusCode, ex.Message)
// 	}
// 	if exist {
// 		return dto.NewFailedResponse[*dto_cm.Conversation](http.StatusBadRequest, "conversation already exists")
// 	}
// 	e := dao_cm.Conversation{
// 		UserId:     u.Id,
// 		Identifier: create.Identifier,
// 		Source:     create.Source,
// 		Channel:    create.Channel,
// 	}
// 	err := dependencies.UnitOfWork.CMConversationRepository().Insert(ctx, &e)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, u.Id, create)
// 		return dto.NewFailedResponse[*dto_cm.Conversation](http.StatusInternalServerError, "error creating conversation")
// 	}
// 	d := dto_cm.NewConversation(e)
// 	return dto.NewSuccessResponse(&d)
// }

// func (Create) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Create a conversation", "A conversation holds the owner and client information and may contain 1 or more conversation messages.", types.HttpRequestTypeJSON, "POST", "/cm/v1/conversations", true, false, types.APITagCM, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("conversation already exists", http.StatusBadRequest)),
// 	})
// }
