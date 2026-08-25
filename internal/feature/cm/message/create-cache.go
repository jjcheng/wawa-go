package feature_cm_message

// import (
// 	"context"
// 	"net/http"

// 	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
// 	dto_core "github.com/jjcheng/wawa-go/internal/dto/core"
// 	"github.com/jjcheng/wawa-go/internal/exception"
// 	"github.com/jjcheng/wawa-go/internal/service"
// )

// type CreateCache struct {
// 	ConversationIdentifier string
// 	Replied                bool
// 	ChatMessages           []dto_core.ChatMessage
// }

// func (createCache CreateCache) Handle(ctx context.Context, dependencies *service.Dependencies) *exception.Exception {
// 	var list []dao_cm.CachedMessage
// 	for _, message := range createCache.ChatMessages {
// 		cachedMessage := dao_cm.CachedMessage{
// 			ConversationIdentifier: createCache.ConversationIdentifier,
// 			Identifier:             message.Identifier,
// 			Role:                   message.Role,
// 			Content:                message.Content,
// 			DateTime:               message.DateTime,
// 			Replied:                createCache.Replied,
// 		}
// 		if message.Attachment != nil {
// 			cachedMessage.AttachmentCaption = message.Attachment.Caption
// 			cachedMessage.AttachmentName = message.Attachment.Name
// 			cachedMessage.AttachmentType = message.Attachment.Type
// 			cachedMessage.AttachmentUrl = message.Attachment.Url
// 		}
// 		if message.Location != nil {
// 			cachedMessage.LocationName = message.Location.Name
// 			cachedMessage.LocationLatitude = message.Location.Latitude
// 			cachedMessage.LocationLongitude = message.Location.Longitude
// 			cachedMessage.LocationAddress = message.Location.Address
// 		}
// 		list = append(list, cachedMessage)
// 	}
// 	err := dependencies.UnitOfWork.CMCachedMessageRepository().InsertBulk(ctx, list)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, createCache)
// 		return exception.NewCustomException("error inserting messages", http.StatusInternalServerError)
// 	}
// 	return nil
// }
