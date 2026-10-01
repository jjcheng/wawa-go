package feature_public

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Ping struct {
}

func (ping *Ping) Validate() []exception.InputException {
	return nil
}

func (ping Ping) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if errors := ping.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](errors)
	}
	if website := getWebsiteFromContext(ctx); website != nil {
		return dto.NewSuccessResponse(website)
	}
	return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found", nil)
}

func (Ping) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Ping a public website",
		"Return the website object based on domain name.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/public/ping",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
