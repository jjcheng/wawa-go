package feature_ai_worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Work struct {
	ConversationId int32 `json:"conversation_id" val:"required" description:"to pull the entire conversation for AI processing"`
}

func (work *Work) Validate() []exception.InputException {
	var errors []exception.InputException
	if work.ConversationId <= 0 {
		errors = append(errors, exception.NewInputException("conversation_id", "missing conversation id"))
	}
	return errors
}

func (work Work) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_worker.WorkResult] {
	if errors := work.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_worker.WorkResult](errors)
	}
	// get conversation
	conversation, err := dependencies.UnitOfWork.AIWorkerConversationRepository().GetById(ctx, work.ConversationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusNotFound, "conversation not found", err)
		}
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// get messages
	messages, err := dependencies.UnitOfWork.AIWorkerMessageRepository().ListByConversationId(ctx, conversation.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// call jev with all apiSettings to detect intent
	messageDTOs := helper.Map(messages, func(m dao_ai_worker.Message) dto_ai_worker.Message {
		return dto_ai_worker.NewMessage(m)
	})
	// get all summaries from all
	apiSettingSummaries := feature.APIIntentDescriptions()
	intents := make([]string, 0, len(apiSettingSummaries))
	for intent := range apiSettingSummaries {
		intents = append(intents, intent)
	}
	questions := map[string]service.TypeSafeChoiceQuestion{
		"detected_intent": {
			Type:         "choice",
			Instructions: "Which intent best matches the latest user message in this conversation?",
			Criteria:     apiSettingSummaries,
		},
	}
	typesafeMessages := make([]service.TypeSafeMessage, 0, len(messageDTOs))
	for i, messageDTO := range messageDTOs {
		typesafeMessages[i] = service.TypeSafeMessage{
			Role:    messageDTO.Role,
			Content: messageDTO.Text(),
		}
	}
	response, err := dependencies.TypeSafe.DetectChoice(ctx, typesafeMessages, questions)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	detectedIndent := response.Answers["detected_intent"]
	// get the apiSetting
	apiSettings, exists := feature.APISettingsBySummary(detectedIndent.Choice)
	if !exists {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusBadRequest, "sorry, we are unable to detect your intent", nil)
	}
	if apiSettings.AIWorker == nil {
		result := dto_ai_worker.NewWorkResult(apiSettings.Summary, "", dto_ai_worker.WorkResultPart{
			Content: "Sorry, this feature is not available to the AI worker yet.",
		})
		return dto.NewSuccessResponse(&result)
	}
	// if not executable, just return the how to message
	if !apiSettings.AIWorker.Executable {
		result := dto_ai_worker.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai_worker.WorkResultPart{
			Content: apiSettings.AIWorker.HowToMessage,
		})
		return dto.NewSuccessResponse(&result)
	}
	// use jev to determine user's action
	var criteria map[string]string = map[string]string{}
	for i, action := range types.AIWorkerActions {
		criteria[fmt.Sprint(i)] = action
	}
	questions = map[string]service.TypeSafeChoiceQuestion{
		"detected_action": {
			Type:         "choice",
			Instructions: fmt.Sprintf("User refers to '%s' feature. Which choice best matches user's latest action in this conversation?", detectedIndent.Choice),
			Criteria:     criteria,
		},
	}
	response, err = dependencies.TypeSafe.DetectChoice(ctx, typesafeMessages, questions)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusBadRequest, types.ExceptionMessageBadGateway, err)
	}
	detectedAction := response.Answers["detected_action"]
	action, exist := criteria[detectedAction.Choice]
	if !exist {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusBadRequest, types.ExceptionMessageBadGateway, nil)
	}
	detectedActionType := types.AIWorkerAction(action)
	switch detectedActionType {
	case types.AIWorkerActionAsk:
		// if asking, present the how to use message
		var message string = apiSettings.AIWorker.HowToMessage
		if apiSettings.AIWorker.Executable {
			inputs, exists := feature.APIRequiredFieldsBySummary(detectedIndent.Choice)
			if !exists {
				return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, nil)
			}
			if len(inputs) == 0 {
				message += "\nDo you want me to pull the data directly here?"
			} else {
				message += "\nOr would you like me to show you the form here?"
			}
		}
		workResult := dto_ai_worker.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai_worker.WorkResultPart{
			Content: message,
		})
		return dto.NewSuccessResponse(&workResult)
	case types.AIWorkerActionReport:
		// if user is reporting an issue using this feature, store in feedback and return sorry message
		workResult := dto_ai_worker.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai_worker.WorkResultPart{
			Content: "we have received your feedback and will work on it soon!",
		})
		return dto.NewSuccessResponse(&workResult)
	}
	// If the user wants to execute this feature, ask for any required request fields first.
	inputs, exists := feature.APIRequiredFieldsBySummary(detectedIndent.Choice)
	if !exists {
		return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, nil)
	}
	// if no required field, execute Handle of the feature and return
	if len(inputs) == 0 {
		result, err := work.executeHandle(detectedIndent.Choice, ctx, user, dependencies)
		if err != nil {
			return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		return dto.NewSuccessResponse(result)
	}
	if len(inputs) > 0 {
		var parts []dto_ai_worker.WorkResultPart
		parts = append(parts, dto_ai_worker.WorkResultPart{
			Content: fmt.Sprintf("To %s, use this form:\n", strings.ToLower(detectedIndent.Choice)),
		})
		// check for any pre-requisits
		if len(apiSettings.AIWorker.Requires) > 0 {
			for _, require := range apiSettings.AIWorker.Requires {
				requiredDataResponse := require.Handler(ctx, user, dependencies)
				if !requiredDataResponse.Success {
					return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, requiredDataResponse.Error)
				}
				parts = append(parts, dto_ai_worker.WorkResultPart{
					Content: require.Title,
				})
				parts = append(parts, RenderFeatureResult(requiredDataResponse.Data, &require.Input))
			}
		}
		for _, input := range inputs {
			// check requires alread have this
			if helper.Any(apiSettings.AIWorker.Requires, func(r feature.AIWorkerRequire) bool {
				return r.Input.ReferenceFieldName == input.Name
			}) {
				continue
			}
			parts = append(parts, dto_ai_worker.WorkResultPart{
				Content: input.Description,
			})
			input.DisplayType = types.AIWorkerDisplayTypeTextbox
			// add inputs
			parts = append(parts, dto_ai_worker.WorkResultPart{
				Content: "",
				Input:   &input,
			})
		}
		result := dto_ai_worker.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, parts...)
		return dto.NewSuccessResponse(&result)
	}
	return dto.NewFailedResponse[*dto_ai_worker.WorkResult](http.StatusNotImplemented, "feature execution is not implemented", nil)
}

func (work *Work) executeHandle(detectedSummary string, ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) (*dto_ai_worker.WorkResult, error) {
	executor, exists := feature.APIExecutorBySummary(detectedSummary)
	if !exists {
		return nil, fmt.Errorf("unable to find the feature by summary")
	}
	response := executor(ctx, user, dependencies, nil)
	if !response.Success {
		result := dto_ai_worker.NewWorkResult(detectedSummary, "", dto_ai_worker.WorkResultPart{
			Content: response.Message,
		})
		return &result, nil
	}
	var parts []dto_ai_worker.WorkResultPart
	parts = append(parts, dto_ai_worker.WorkResultPart{
		Content: detectedSummary,
	})
	parts = append(parts, RenderFeatureResult(response.Data, nil))
	result := dto_ai_worker.NewWorkResult(detectedSummary, "", parts...)
	return &result, nil
}

func RenderFeatureResult(data any, input *dto_ai_worker.WorkInput) dto_ai_worker.WorkResultPart {
	value := reflect.ValueOf(data)
	value = indirectValue(value)
	if !value.IsValid() || isEmptyResult(value) {
		return textResultPart("Sorry, we cannot find any result.")
	}
	if value.Kind() == reflect.Struct {
		if items := value.FieldByName("Items"); items.IsValid() && (items.Kind() == reflect.Slice || items.Kind() == reflect.Array) {
			return RenderFeatureResult(items.Interface(), input)
		}
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return renderFeatureTable(value, input)
	case reflect.Struct:
		return renderFeatureObject(value)
	default:
		return textResultPart(formatFeatureValue(value))
	}
}

func isEmptyResult(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	default:
		return false
	}
}

func renderFeatureTable(rows reflect.Value, input *dto_ai_worker.WorkInput) dto_ai_worker.WorkResultPart {
	if rows.Len() == 0 {
		return textResultPart("Sorry, we cannot find any result.")
	}
	first := indirectValue(rows.Index(0))
	if !first.IsValid() || first.Kind() != reflect.Struct {
		items := make([][]any, 0, rows.Len())
		for index := 0; index < rows.Len(); index++ {
			items = append(items, []any{valueInterface(rows.Index(index))})
		}
		part := dto_ai_worker.WorkResultPart{Content: items, Input: input}
		return part
	}

	columns := titledFields(first.Type())
	if len(columns) == 0 {
		return textResultPart(formatFeatureValue(rows))
	}

	titles := make([]string, 0, len(columns))
	fieldNames := make([]string, 0, len(columns))
	for _, column := range columns {
		titles = append(titles, column.title)
		fieldNames = append(fieldNames, column.fieldName)
	}
	content := make([][]any, 0, rows.Len())
	for rowIndex := 0; rowIndex < rows.Len(); rowIndex++ {
		row := indirectValue(rows.Index(rowIndex))
		if !row.IsValid() || row.Kind() != reflect.Struct {
			continue
		}
		values := make([]any, 0, len(columns))
		for _, column := range columns {
			values = append(values, valueInterface(fieldByIndex(row, column.index)))
		}
		content = append(content, values)
	}
	part := dto_ai_worker.WorkResultPart{
		FieldNames: fieldNames,
		Titles:     titles,
		Content:    content,
		Input:      input,
	}
	return part
}

func renderFeatureObject(object reflect.Value) dto_ai_worker.WorkResultPart {
	fields := titledFields(object.Type())
	if len(fields) == 0 {
		return textResultPart(formatFeatureValue(object))
	}
	titles := make([]string, 0, len(fields))
	content := make([]any, 0, len(fields))
	for _, field := range fields {
		titles = append(titles, field.title)
		content = append(content, valueInterface(fieldByIndex(object, field.index)))
	}
	return dto_ai_worker.WorkResultPart{
		Titles:  titles,
		Content: content,
	}
}

func textResultPart(message string) dto_ai_worker.WorkResultPart {
	return dto_ai_worker.WorkResultPart{Content: message}
}

type titledField struct {
	index     []int
	fieldName string
	title     string
}

func titledFields(structType reflect.Type) []titledField {
	return collectTitledFields(structType, nil, make(map[reflect.Type]bool))
}

func collectTitledFields(structType reflect.Type, parentIndex []int, visited map[reflect.Type]bool) []titledField {
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct || visited[structType] {
		return nil
	}
	visited[structType] = true
	defer delete(visited, structType)

	fields := make([]titledField, 0, structType.NumField())
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		fieldIndex := append(append([]int(nil), parentIndex...), index)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous {
			fields = append(fields, collectTitledFields(field.Type, fieldIndex, visited)...)
			continue
		}
		if title := field.Tag.Get("title"); title != "" {
			fields = append(fields, titledField{index: fieldIndex, fieldName: fieldTagName(field), title: title})
		}
	}
	return fields
}

func fieldTagName(field reflect.StructField) string {
	for _, tagName := range []string{"json", "form", "uri"} {
		name := strings.Split(field.Tag.Get(tagName), ",")[0]
		if name != "" && name != "-" {
			return name
		}
	}
	return field.Name
}

func fieldByIndex(value reflect.Value, index []int) reflect.Value {
	for _, fieldIndex := range index {
		value = indirectValue(value)
		if !value.IsValid() || value.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		value = value.Field(fieldIndex)
	}
	return value
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func formatFeatureValue(value reflect.Value) string {
	value = indirectValue(value)
	if !value.IsValid() {
		return ""
	}
	if value.Kind() == reflect.Struct || value.Kind() == reflect.Map || value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		encoded, err := json.Marshal(value.Interface())
		if err == nil {
			return string(encoded)
		}
	}
	return fmt.Sprint(value.Interface())
}

func valueInterface(value reflect.Value) any {
	value = indirectValue(value)
	if !value.IsValid() {
		return nil
	}
	return value.Interface()
}
