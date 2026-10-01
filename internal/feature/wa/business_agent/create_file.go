package feature_wa_business_agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type CreateFile struct {
	PhoneNumberId int32  `form:"phone_number_id" val:"required" description:"id of the phone number"`
	FileName      string `form:"file_name" val:"required" description:"file name registered with the agent"`
	Content       []byte `json:"-"`
}

func (createFile *CreateFile) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if createFile.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	createFile.FileName = strings.TrimSpace(createFile.FileName)
	if createFile.FileName == "" {
		inputErrors = append(inputErrors, exception.NewInputException("file_name", "missing file name"))
	}
	if len(createFile.Content) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("file", "file content is required"))
	}
	return inputErrors
}

func (createFile CreateFile) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentFile] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentFile](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentFile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := createFile.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentFile](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, createFile.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentFile](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentFile](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == createFile.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentFile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentFile](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	agentFile, err := dependencies.Facebook.CreateAgentFile(ctx, phoneNumber.MetaPhoneNumberId, createFile.FileName, createFile.Content, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentFile](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(agentFile)
}

func (CreateFile) APISettings() feature.APISettings {
	settings := feature.NewAPISettings(
		"Create WhatsApp business agent file",
		"Uploads a file to a WhatsApp business agent using multipart form data.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/business-agent/files",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
	settings.BodyContentType = "multipart/form-data"
	return settings
}
