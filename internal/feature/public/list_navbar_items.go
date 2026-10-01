package feature_public

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListNavbarItems struct {
}

func (ListNavbarItems) Validate() []exception.InputException {
	return nil
}

func (listNavbarItems ListNavbarItems) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_commerce.NavBarItem] {
	if inputErrors := listNavbarItems.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_commerce.NavBarItem](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[[]dto_commerce.NavBarItem](http.StatusNotFound, "website not found", nil)
	}
	pages, err := dependencies.UnitOfWork.CommercePageRepository().ListByOnNavBar(ctx, website.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_commerce.NavBarItem](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var navBarItems []dto_commerce.NavBarItem
	for _, page := range pages {
		navBarItems = append(navBarItems, dto_commerce.NavBarItem{
			Title: page.Title,
			Slug:  fmt.Sprintf("/%s", page.Slug),
		})
	}
	return dto.NewSuccessResponse(navBarItems)
}

func (ListNavbarItems) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List public website navigation items",
		"List all pages shown in a public website navigation bar.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/public/navbar-items",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
