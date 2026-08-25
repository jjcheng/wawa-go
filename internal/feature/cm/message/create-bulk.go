package feature_cm_message

// import (
// 	"context"
// 	"fmt"
// 	"net/http"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	"github.com/jjcheng/wawa-go/internal/dto"
// 	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
// 	dto_cm "github.com/jjcheng/wawa-go/internal/dto/cm"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/repository"
// 	"github.com/jjcheng/wawa-go/internal/service"
// )

// type CreateBulk struct {
// 	List        []Create              `json:"list" val:"required" description:"list of conversation messages to insert"`
// 	Transaction repository.UnitOfWork `json:"-"` // to rollback if failed
// }

// func (createBulk *CreateBulk) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if len(createBulk.List) == 0 {
// 		return nil
// 	} else {
// 		for i := range createBulk.List {
// 			er := createBulk.List[i].Validate()
// 			if len(er) > 0 {
// 				for _, e := range er {
// 					errors = append(errors, exception.NewInputException(fmt.Sprintf("list[%d].%s", i, e.Field), e.Message))
// 				}
// 			}
// 		}
// 	}
// 	return errors
// }

// func (createBulk CreateBulk) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[[]dto_cm.Message] {
// 	if len(createBulk.List) == 0 {
// 		return dto.NewSuccessResponse([]dto_cm.Message{})
// 	}
// 	// validate
// 	if errors := createBulk.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[[]dto_cm.Message](errors)
// 	}
// 	es := []dao_cm.Message{}
// 	for _, item := range createBulk.List {
// 		e := dao_cm.Message{
// 			ConversationId: item.ConversationId,
// 			Identifier:     item.Identifier,
// 			SessionId:      item.SessionId,
// 			Role:           item.Role,
// 			Content:        item.Content,
// 			DateTime:       item.DateTime,
// 		}
// 		if item.Attachment != nil {
// 			e.AttachmentName = item.Attachment.Name
// 			e.AttachmentType = item.Attachment.Type
// 			e.AttachmentUrl = item.Attachment.Url
// 			e.AttachmentCaption = item.Attachment.Caption
// 		}
// 		if item.Location != nil {
// 			e.LocationName = item.Location.Name
// 			e.LocationLatitude = item.Location.Latitude
// 			e.LocationLongitude = item.Location.Longitude
// 			e.LocationAddress = item.Location.Address
// 		}
// 		es = append(es, e)
// 	}
// 	// this process has to be wrapped by transaction, anything goes wrong has to abort the whole thing
// 	transaction := createBulk.Transaction
// 	// if transaction comes from external, commit and rollback will be handled externally
// 	isOwnTransaction := false
// 	if transaction == nil {
// 		transaction = dependencies.UnitOfWork.BeginTransaction()
// 		isOwnTransaction = true
// 	}
// 	err := transaction.CMMessageRepository().InsertBulk(ctx, es)
// 	if err != nil {
// 		if isOwnTransaction {
// 			transaction.Rollback()
// 		}
// 		dependencies.Logger.ErrorFunction(err, user.Id, createBulk)
// 		return dto.NewFailedResponse[[]dto_cm.Message](http.StatusInternalServerError, "error creating messages")
// 	}
// 	if isOwnTransaction {
// 		transaction.CommitTransaction()
// 	}
// 	ds := []dto_cm.Message{}
// 	for _, e := range es {
// 		ds = append(ds, dto_cm.NewMessage(e))
// 	}
// 	return dto.NewSuccessResponse(ds)
// }
