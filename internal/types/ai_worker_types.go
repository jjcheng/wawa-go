package types

type AIWorkerMessageRole string

const (
	AIWorkerMessageRoleUser      AIWorkerMessageRole = "USER"
	AIWorkerMessageRoleAssistant AIWorkerMessageRole = "ASSISTANT"
)

type AIWorkerAction string

const (
	AIWorkerActionAsk     AIWorkerAction = "User is asking how to use this feature."
	AIWorkerActionReport  AIWorkerAction = "User is having problem using this feature."
	AIWorkerActionExecute AIWorkerAction = "User wants to use this feature."
)

var AIWorkerActions []string = []string{
	string(AIWorkerActionAsk),
	string(AIWorkerActionReport),
	string(AIWorkerActionExecute),
}

type AIWorkResultPartType string

const (
	AIWorkResultPartTypeTitle AIWorkResultPartType = "TITLE"
	AIWorkResultPartTypeText  AIWorkResultPartType = "TEXT"
	AIWorkResultPartTypeData  AIWorkResultPartType = "DATA"
)

type AIWorkerInputStatus string

const (
	AIWorkerInputStatusComplete   AIWorkerInputStatus = "Yes, all required inputs are provided by user."
	AIWorkerInputStatusIncomplete AIWorkerInputStatus = "No, some required inputs are missing."
	AIWorkerInputStatusEmpty      AIWorkerInputStatus = "No, user did not provide any of the required inputs."
)

var AIWorkerInputStatuss []string = []string{
	string(AIWorkerInputStatusComplete),
	string(AIWorkerInputStatusIncomplete),
	string(AIWorkerInputStatusEmpty),
}

type AIWorkerInputType string

const (
	AIWorkerInputFieldTypeText     AIWorkerInputType = "TEXT"
	AIWorkerInputFieldTypeInt      AIWorkerInputType = "INT"
	AIWorkerInputFieldTypeFloat    AIWorkerInputType = "FLOAT"
	AIWorkerInputFieldTypeDate     AIWorkerInputType = "DATE"
	AIWorkerInputFieldTypeDateTime AIWorkerInputType = "DATETIME"
	AIWorkerInputFieldTypeBool     AIWorkerInputType = "BOOL"
	AIWorkerInputFieldTypeSelect   AIWorkerInputType = "SELECT"
)

type AIWorkerReturnType string

const (
	AIWorkerReturnTypeText AIWorkerReturnType = "TEXT"
	AIWorkerReturnTypeData AIWorkerReturnType = "DATA"
)

type AIWorkerDisplayType string

const (
	AIWorkerDisplayTypeTextbox           AIWorkerDisplayType = "TEXTBOX"
	AIWorkerDisplayTypeTextarea          AIWorkerDisplayType = "TEXTAREA"
	AIWorkerDisplayTypeReadonlyTable     AIWorkerDisplayType = "READONLY_TABLE"
	AIWorkerDisplayTypeSingleChoiceTable AIWorkerDisplayType = "SINGLE_CHOICE_TABLE"
	AIWorkerDisplayTypeMultiChoiceTable  AIWorkerDisplayType = "MULTI_CHOICE_TABLE"
)
