package feature_ai_conversation

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Execute struct {
	Feature string         `json:"feature"`
	Form    map[string]any `json:"form"`
}

func (execute *Execute) Validate() []exception.InputException {
	execute.Feature = strings.TrimSpace(execute.Feature)
	var errors []exception.InputException
	if execute.Feature == "" {
		errors = append(errors, exception.NewInputException("feature", "missing feature"))
	}
	if execute.Form == nil {
		errors = append(errors, exception.NewInputException("form", "missing form element"))
	}
	return errors
}

func (execute Execute) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.WorkResult] {
	if inputErrors := execute.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.WorkResult](inputErrors)
	}
	settings, exists := feature.APISettingsBySummary(execute.Feature)
	if !exists {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadRequest, "unknown feature", nil)
	}
	if settings.AIWorker == nil || !settings.AIWorker.Executable {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusForbidden, "feature is not executable by the AI worker", nil)
	}
	executor, exists := feature.APIExecutorBySummary(execute.Feature)
	if !exists {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, nil)
	}
	featureResponse := executor(ctx, user, dependencies, execute.Form)
	if !featureResponse.Success {
		return dto.Response[*dto_ai.WorkResult]{
			ResponseBase: featureResponse.ResponseBase,
			Error:        featureResponse.Error,
		}
	}
	result := dto_ai.NewWorkResult(execute.Feature, "",
		dto_ai.WorkResultPart{Type: types.AIWorkResultPartTypeText, Content: "Here is the data you requested:"},
		renderFeatureResult(featureResponse.Data, nil),
	)
	return dto.NewSuccessResponse(&result)
}

func (Execute) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Execute an AI worker feature",
		"Execute an AI-enabled feature using the supplied form values.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/ai/conversations/execute",
		true,
		true,
		types.APITagAI,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("unknown feature", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("feature is not executable by the AI worker", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
