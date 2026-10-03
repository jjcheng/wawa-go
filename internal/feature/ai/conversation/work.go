package feature_ai_conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
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

func (work Work) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.WorkResult] {
	if errors := work.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.WorkResult](errors)
	}
	// get conversation
	conversation, err := dependencies.UnitOfWork.AIConversationRepository().GetById(ctx, work.ConversationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusNotFound, "conversation not found", err)
		}
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if conversation.UserId != user.Id {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// get messages
	messages, err := dependencies.UnitOfWork.AIMessageRepository().ListByConversationId(ctx, conversation.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// call jev with all apiSettings to detect intent
	messageDTOs := helper.Map(messages, func(m dao_ai.Message) dto_ai.Message {
		return dto_ai.NewMessage(m)
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
	response, err := dependencies.TypeSafe.DetectChoice(ctx, messageDTOs, questions)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	detectedIndent := response.Answers["detected_intent"]
	// get the apiSetting
	apiSettings, exists := feature.APISettingsBySummary(detectedIndent.Choice)
	if !exists {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadRequest, "sorry, we are unable to detect your intent", nil)
	}
	if apiSettings.AIWorker == nil {
		result := dto_ai.NewWorkResult(apiSettings.Summary, "", dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
			Content: "Sorry, this feature is not available to the AI worker yet.",
		})
		return dto.NewSuccessResponse(&result)
	}
	// if not executable, just return the how to message
	if !apiSettings.AIWorker.Executable {
		result := dto_ai.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
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
	response, err = dependencies.TypeSafe.DetectChoice(ctx, messageDTOs, questions)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadRequest, types.ExceptionMessageBadGateway, err)
	}
	detectedAction := response.Answers["detected_action"]
	action, exist := criteria[detectedAction.Choice]
	if !exist {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadRequest, types.ExceptionMessageBadGateway, nil)
	}
	detectedActionType := types.AIWorkerAction(action)
	switch detectedActionType {
	case types.AIWorkerActionAsk:
		// if asking, present the how to use message
		var message string = apiSettings.AIWorker.HowToMessage
		if apiSettings.AIWorker.Executable {
			message += "\nOr would you like me show you the form here?"
		}
		workResult := dto_ai.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
			Content: message,
		})
		return dto.NewSuccessResponse(&workResult)
	case types.AIWorkerActionReport:
		// if user is reporting an issue using this feature, store in feedback and return sorry message
		workResult := dto_ai.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
			Content: "we have received your feedback and will work on it soon!",
		})
		return dto.NewSuccessResponse(&workResult)
	}
	// If the user wants to execute this feature, ask for any required request fields first.
	requiredFields, exists := feature.APIRequiredFieldsBySummary(detectedIndent.Choice)
	if !exists {
		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, nil)
	}
	// if no required field, execute Handle of the feature and return
	if len(requiredFields) == 0 {
		result, err := work.executeHandle(detectedIndent.Choice, ctx, user, dependencies)
		if err != nil {
			return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		return dto.NewSuccessResponse(result)
	}
	if len(requiredFields) > 0 {
		var parts []dto_ai.WorkResultPart
		parts = append(parts, dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
			Content: fmt.Sprintf("To %s, please fill up this form:\n", strings.ToLower(detectedIndent.Choice)),
		})
		// check for any pre-requisits
		if len(apiSettings.AIWorker.Requires) > 0 {
			for _, require := range apiSettings.AIWorker.Requires {
				requiredDataResponse := require.Handler(ctx, user, dependencies)
				if !requiredDataResponse.Success {
					return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, requiredDataResponse.Error)
				}
				parts = append(parts, dto_ai.WorkResultPart{
					Type:    types.AIWorkResultPartTypeText,
					Content: require.Title,
					Input:   &require.Input,
				})
				parts = append(parts, renderFeatureResult(requiredDataResponse.Data, &require.Type))
			}
		}
		for _, input := range requiredFields {
			parts = append(parts, dto_ai.WorkResultPart{
				Type:    types.AIWorkResultPartTypeText,
				Content: input.Description,
			})
			// add inputs
			parts = append(parts, dto_ai.WorkResultPart{
				Content: "",
				Input:   &input,
			})
		}
		// criteria = map[string]string{}
		// for i := range types.AIWorkerInputStatuss {
		// 	criteria[fmt.Sprint(i)] = types.AIWorkerInputStatuss[i]
		// }
		// questions = map[string]service.TypeSafeChoiceQuestion{
		// 	"user_input_status": {
		// 		Type:         "noul",
		// 		Instructions: fmt.Sprintf("User refers to %s feature. The required inputs from the user are:\n%s\nDid user provide all the required inputs?", detectedIndent.Choice, strings.Join(requiredInputs, "\n")),
		// 		Criteria: map[string]string{
		// 			"true":  "All required inputs are provided by the user",
		// 			"false": "Some or all required inputs are not provided by the user",
		// 		},
		// 	},
		// }
		// questions = map[string]service.TypeSafeChoiceQuestion{
		// 	"user_input_status": {
		// 		Type:         "choice",
		// 		Instructions: fmt.Sprintf("User refers to %s feature. The required inputs from the user are:\n%s\nDid user provide all the required inputs?", detectedIndent.Choice, strings.Join(requiredInputs, "\n")),
		// 		Criteria:     criteria,
		// 	},
		// }
		// detectedNoul, err := dependencies.TypeSafe.DetectChoice(ctx, messageDTOs, questions)
		// if err != nil {
		// 	return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
		// }
		// noul := detectedNoul.Answers["user_input_status"]
		// answers, err := dependencies.TypeSafe.DetectChoice(ctx, messageDTOs, questions)
		// if err != nil {
		// 	return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
		// }
		// answersAnswer := answers.Answers["user_input_status"]
		// userInputStatus := types.AIWorkerInputStatus(criteria[answersAnswer.Choice])
		//var userInputStatus types.AIWorkerInputStatus
		// if noul.Noul > 0.5 {
		// 	userInputStatus = types.AIWorkerInputStatusComplete
		// } else {
		// 	userInputStatus = types.AIWorkerInputStatusIncomplete
		// }
		// switch userInputStatus {
		// case types.AIWorkerInputStatusComplete:
		// 	// TODO: extract entities
		// 	// execute
		// 	result, err := work.executeHandle(detectedIndent.Choice, ctx, user, dependencies)
		// 	if err != nil {
		// 		return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		// 	}
		// 	return dto.NewSuccessResponse(result)
		// case types.AIWorkerInputStatusEmpty:
		// 	// parts = append(parts, dto_ai.WorkResultPart{
		// 	// 	Type:    types.AIWorkResultPartTypeText,
		// 	// 	Content: fmt.Sprintf("In order to execute your instruction, I need your follow inputs:\n%s", strings.Join(requiredInputs, "\n")),
		// 	// })
		// 	result := dto_ai.NewWorkResult(parts...)
		// 	return dto.NewSuccessResponse(&result)
		// default:
		// 	// parts = append(parts, dto_ai.WorkResultPart{
		// 	// 	Type:    types.AIWorkResultPartTypeText,
		// 	// 	Content: fmt.Sprintf("You have missed some inputs. Please provide all of follow inputs:\n%s", strings.Join(requiredInputs, "\n")),
		// 	// })
		// 	result := dto_ai.NewWorkResult(parts...)
		// 	return dto.NewSuccessResponse(&result)
		// }
		result := dto_ai.NewWorkResult(apiSettings.Summary, apiSettings.AIWorker.URL, parts...)
		return dto.NewSuccessResponse(&result)
	}
	return dto.NewFailedResponse[*dto_ai.WorkResult](http.StatusNotImplemented, "feature execution is not implemented", nil)
}

func (work *Work) executeHandle(detectedSummary string, ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) (*dto_ai.WorkResult, error) {
	executor, exists := feature.APIExecutorBySummary(detectedSummary)
	if !exists {
		return nil, fmt.Errorf("unable to find the feature by summary")
	}
	response := executor(ctx, user, dependencies, nil)
	if !response.Success {
		result := dto_ai.NewWorkResult(detectedSummary, "", dto_ai.WorkResultPart{
			Type:    types.AIWorkResultPartTypeText,
			Content: response.Message,
		})
		return &result, nil
	}
	var parts []dto_ai.WorkResultPart
	parts = append(parts, dto_ai.WorkResultPart{
		Type:    types.AIWorkResultPartTypeText,
		Content: "Here is the data you have requested:",
	})
	parts = append(parts, renderFeatureResult(response.Data, nil))
	result := dto_ai.NewWorkResult(detectedSummary, "", parts...)
	return &result, nil
}

func renderFeatureResult(data any, typ *types.AIWorkResultPartType) dto_ai.WorkResultPart {
	value := reflect.ValueOf(data)
	value = indirectValue(value)
	if !value.IsValid() || isEmptyResult(value) {
		return textResultPart("Sorry, we cannot find any result.")
	}

	if value.Kind() == reflect.Struct {
		if items := value.FieldByName("Items"); items.IsValid() && (items.Kind() == reflect.Slice || items.Kind() == reflect.Array) {
			return renderFeatureResult(items.Interface(), typ)
		}
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return renderFeatureTable(value, typ)
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

func renderFeatureTable(rows reflect.Value, typ *types.AIWorkResultPartType) dto_ai.WorkResultPart {
	if rows.Len() == 0 {
		return textResultPart("Sorry, we cannot find any result.")
	}
	first := indirectValue(rows.Index(0))
	if !first.IsValid() || first.Kind() != reflect.Struct {
		items := make([][]any, 0, rows.Len())
		for index := 0; index < rows.Len(); index++ {
			items = append(items, []any{valueInterface(rows.Index(index))})
		}
		part := dto_ai.WorkResultPart{Type: types.AIWorkResultPartTypeList, Content: items}
		if typ != nil {
			part.Type = *typ
		}
		return part
	}

	columns := titledFields(first.Type())
	if len(columns) == 0 {
		return textResultPart(formatFeatureValue(rows))
	}

	titles := make([]string, 0, len(columns))
	for _, column := range columns {
		titles = append(titles, column.title)
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
	part := dto_ai.WorkResultPart{
		Type:    types.AIWorkResultPartTypeList,
		Titles:  titles,
		Content: content,
	}
	if typ != nil {
		part.Type = *typ
	}
	return part
}

func renderFeatureObject(object reflect.Value) dto_ai.WorkResultPart {
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
	return dto_ai.WorkResultPart{
		Type:    types.AIWorkResultPartTypeObject,
		Titles:  titles,
		Content: content,
	}
}

func textResultPart(message string) dto_ai.WorkResultPart {
	return dto_ai.WorkResultPart{Type: types.AIWorkResultPartTypeText, Content: message}
}

type titledField struct {
	index []int
	title string
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
			fields = append(fields, titledField{index: fieldIndex, title: title})
		}
	}
	return fields
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
