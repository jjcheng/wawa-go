package feature

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type embeddedRequiredRequestFields struct {
	PhoneNumberId int32 `json:"phone_number_id" val:"required" description:"phone number id"`
}

type requiredRequestFields struct {
	embeddedRequiredRequestFields
	UserId      int32     `json:"user_id,omitempty" val:"required" description:"user id"`
	PhoneNumber string    `form:"phone_number" val:"required" description:"phone number"`
	ResourceId  int32     `uri:"resource_id" val:"required" description:"resource id"`
	Ratio       float64   `json:"ratio" val:"required" description:"ratio value"`
	ScheduleAt  time.Time `json:"schedule_at" val:"required" description:"schedule time"`
	Optional    string    `json:"optional" val:"omitempty" description:"optional value"`
	Ignored     string    `json:"ignored" val:"notrequired" description:"not required"`
}

func TestRequiredFieldDescriptions(t *testing.T) {
	got := requiredFieldDescriptions(reflect.TypeFor[*requiredRequestFields](), make(map[reflect.Type]bool))
	want := []dto_ai.WorkInput{
		{Name: "phone_number_id", Description: "phone number id", Type: types.AIInputFieldTypeInt},
		{Name: "user_id", Description: "user id", Type: types.AIInputFieldTypeInt},
		{Name: "phone_number", Description: "phone number", Type: types.AIInputFieldTypeText},
		{Name: "resource_id", Description: "resource id", Type: types.AIInputFieldTypeInt},
		{Name: "ratio", Description: "ratio value", Type: types.AIInputFieldTypeFloat},
		{Name: "schedule_at", Description: "schedule time", Type: types.AIInputFieldTypeDateTime},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("requiredFieldDescriptions() = %v, want %v", got, want)
	}
}

type executorTestRequest struct {
	Value string `json:"value"`
}

func (executorTestRequest) APISettings() APISettings {
	return APISettings{}
}

func (request executorTestRequest) Handle(context.Context, *dto_account.User, *service.Dependencies) dto.Response[string] {
	return dto.NewSuccessResponse(request.Value)
}

func TestRegisteredAPIExecutorAdaptsTypedResponse(t *testing.T) {
	const summary = "test executor typed response"
	RegisterAPIExecutor(summary, executorTestRequest{})

	executor, exists := APIExecutorBySummary(summary)
	if !exists {
		t.Fatal("APIExecutorBySummary() did not find registered executor")
	}
	response := executor(context.Background(), nil, nil, map[string]any{"value": "executed"})
	if !response.Success || response.Data != "executed" {
		t.Fatalf("executor response = %+v, want successful response with decoded form value %q", response, "executed")
	}
}
