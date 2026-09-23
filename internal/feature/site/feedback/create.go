package feature_site_feedback

import (
	"context"
	"net/http"
	"strings"

	dao_site "github.com/jjcheng/wawa-go/internal/dao/site"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_site "github.com/jjcheng/wawa-go/internal/dto/site"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Create struct {
	Content string `json:"content" val:"required" description:"content of the feedback"`
}

func (create *Create) Validate() []exception.InputException {
	create.Content = strings.TrimSpace(create.Content)
	inputErrors := []exception.InputException{}
	if create.Content == "" {
		inputErrors = append(inputErrors, exception.NewInputException("content", "missing content"))
	} else if len(create.Content) < 20 {
		inputErrors = append(inputErrors, exception.NewInputException("content", "content must have at least 20 characters"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_site.Feedback] {
	if user == nil {
		return dto.NewFailedResponse[*dto_site.Feedback](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_site.Feedback](inputErrors)
	}
	feedback := dao_site.Feedback{
		UserId:  user.Id,
		Content: create.Content,
	}
	if err := dependencies.UnitOfWork.SiteFeedbackRepository().Insert(ctx, &feedback); err != nil {
		return dto.NewFailedResponse[*dto_site.Feedback](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_site.NewFeedback(feedback)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create site feedback",
		"Creates feedback from the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/site/feedback",
		true,
		true,
		types.APITagSite,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
