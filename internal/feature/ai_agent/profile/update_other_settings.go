package feature_ai_agent_profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UpdateOtherSettings struct {
	Id              int32    `uri:"id" val:"required" description:"id of the profile"`
	HandoverMessage string   `json:"handover_message" description:"handover_message"`
	NeverSayPhrases []string `json:"never_say_phrases" description:"never_say_phrases"`
}

func (update *UpdateOtherSettings) Validate() []exception.InputException {
	if update.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid profile id")}
	}
	return nil
}

func (update UpdateOtherSettings) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, update.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	profile.HandoverMessage = update.HandoverMessage
	profile.NeverSayPhrases = pq.StringArray(update.NeverSayPhrases)
	if err := dependencies.UnitOfWork.AIAgentProfileRepository().Update(ctx, profile); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse[any](nil)
}

func (UpdateOtherSettings) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update AI agent profile other settings",
		"Replaces the profile handover message and never-say phrases. Empty or omitted values clear the corresponding setting.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/ai-agent/profiles/:id/other-settings",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
