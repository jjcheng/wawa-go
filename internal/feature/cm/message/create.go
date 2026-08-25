package feature_cm_message

// import (
// 	"context"
// 	"net/http"
// 	"strings"
// 	"time"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	dto_core "github.com/jjcheng/wawa-go/internal/dto/core"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/service"
// 	"github.com/jjcheng/wawa-go/internal/types"
// )

// type Create struct {
// 	ConversationId   int32                           `json:"conversation_id" val:"required" description:"id of the conversation, get from conversation" example:"1"`
// 	SessionId        string                          `json:"session_id" val:"required" description:"an uuid to track the session of 1 request" conversation:"16fd2706-8baf-433b-82eb-8c7fada847da"`
// 	Identifier       string                          `json:"identifier" val:"required" description:"uuid, for mobi to track the status, must be unique for each message" example:"16fd2706-8baf-433b-82eb-8c7fada847da"`
// 	Role             types.ChatMessageRole           `json:"role" val:"required" description:"USER, ASSISTANT, GREET, WAIT" example:"USER"`
// 	Content          string                          `json:"content" val:"required" description:"the content of the message" example:"hi"`
// 	DateTime         time.Time                       `json:"date_time" val:"required" description:"datetime this message is created" example:"2025-10-01T08:08:00Z"`
// 	Attachment       *dto_core.ChatMessageAttachment `json:"attachment" description:"any attachment"`
// 	Location         *dto_core.ChatMessageLocation   `json:"location" description:"any location"`
// 	ParentIdentifier string                          `json:"parent_identifier"`
// }

// func (create *Create) Validate() []exception.InputException {
// 	create.SessionId = strings.TrimSpace(create.SessionId)
// 	create.Identifier = strings.TrimSpace(create.Identifier)
// 	create.Content = strings.TrimSpace(create.Content)
// 	create.ParentIdentifier = strings.TrimSpace(create.ParentIdentifier)
// 	errors := []exception.InputException{}
// 	if create.ConversationId <= 0 {
// 		errors = append(errors, exception.NewInputException("conversation_id", "missing conversation_id"))
// 	}
// 	if create.Identifier == "" {
// 		errors = append(errors, exception.NewInputException("identifier", "missing identifier"))
// 	}
// 	if create.SessionId == "" {
// 		errors = append(errors, exception.NewInputException("session_id", "missing session_id"))
// 	}
// 	if create.Role == "" {
// 		errors = append(errors, exception.NewInputException("role", "missing role"))
// 	}
// 	if create.Content == "" {
// 		errors = append(errors, exception.NewInputException("content", "missing content"))
// 	}
// 	if create.DateTime.IsZero() {
// 		errors = append(errors, exception.NewInputException("date_time", "missing date_time"))
// 	}
// 	if create.Attachment != nil {
// 		errors = append(errors, create.Attachment.Validate()...)
// 	}
// 	if create.Location != nil {
// 		errors = append(errors, create.Location.Validate()...)
// 	}
// 	return errors
// }

// func (create Create) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[*dto_cm.Message] {
// 	if errors := create.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_cm.Message](errors)
// 	}
// 	e := dao_cm.Message{
// 		ConversationId:   create.ConversationId,
// 		SessionId:        create.SessionId,
// 		Identifier:       create.Identifier,
// 		Role:             create.Role,
// 		Content:          create.Content,
// 		DateTime:         create.DateTime,
// 		ParentIdentifier: create.ParentIdentifier,
// 	}
// 	if create.Attachment != nil {
// 		e.AttachmentName = create.Attachment.Name
// 		e.AttachmentType = create.Attachment.Type
// 		e.AttachmentUrl = create.Attachment.Url
// 		e.AttachmentCaption = create.Attachment.Caption
// 	}
// 	if create.Location != nil {
// 		e.LocationName = create.Location.Name
// 		e.LocationLatitude = create.Location.Latitude
// 		e.LocationLongitude = create.Location.Longitude
// 		e.LocationAddress = create.Location.Address
// 	}
// 	err := dependencies.UnitOfWork.CMMessageRepository().Insert(ctx, &e)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, create)
// 		return dto.NewFailedResponse[*dto_cm.Message](http.StatusInternalServerError, "error creating message")
// 	}
// 	d := dto_cm.NewMessage(e)
// 	return dto.NewSuccessResponse(&d)
// }
