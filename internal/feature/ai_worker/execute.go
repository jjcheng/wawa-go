package feature_ai_worker

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
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

func (execute Execute) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_worker.WorkResult] {
	if inputErrors := execute.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_worker.WorkResult](inputErrors)
	}
	apiSettings, exists := feature.APISettingsBySummary(execute.Feature)
	if !exists {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusBadRequest, "unknown feature", nil)
	}
	if apiSettings.AIWorker == nil || !apiSettings.AIWorker.Executable {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusForbidden, "feature is not executable by the AI worker", nil)
	}
	executor, exists := feature.APIExecutorBySummary(execute.Feature)
	if !exists {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, nil)
	}
	featureResponse := executor(ctx, user, dependencies, execute.Form)
	if !featureResponse.Success {
		return dto.Response[*dto_ai_worker.WorkResult]{
			ResponseBase: featureResponse.ResponseBase,
			Error:        featureResponse.Error,
		}
	}
	var result dto_ai_worker.WorkResult
	if apiSettings.AIWorker.ReturnType == types.AIWorkerReturnTypeText {
		result = dto_ai_worker.NewWorkResult(execute.Feature, "",
			dto_ai_worker.WorkResultPart{Content: apiSettings.AIWorker.ReturnText, Color: "green"},
		)
	} else {
		result = dto_ai_worker.NewWorkResult(execute.Feature, "",
			dto_ai_worker.WorkResultPart{Content: "Here is the data you have requested:"},
			RenderFeatureResult(featureResponse.Data, nil),
		)
	}
	return dto.NewSuccessResponse(&result)
}

func (Execute) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Execute an AI worker feature",
		"Execute an AI-enabled feature using the supplied form values.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/ai/worker/execute",
		true,
		true,
		types.APITagAIWorker,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("unknown feature", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("feature is not executable by the AI worker", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
