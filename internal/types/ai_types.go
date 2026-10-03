package types

type AIMessageRole string

const (
	AIMessageRoleUser      AIMessageRole = "USER"
	AIMessageRoleAssistant AIMessageRole = "ASSISTANT"
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
	AIWorkResultPartTypeText    AIWorkResultPartType = "TEXT"
	AIWorkerResultPartTypeInput AIWorkResultPartType = "INPUT"
	AIWorkResultPartTypeObject  AIWorkResultPartType = "OBJECT"
	AIWorkResultPartTypeList    AIWorkResultPartType = "LIST"
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

type AIInputType string

const (
	AIInputFieldTypeText     AIInputType = "TEXT"
	AIInputFieldTypeInt      AIInputType = "INT"
	AIInputFieldTypeFloat    AIInputType = "FLOAT"
	AIInputFieldTypeDate     AIInputType = "DATE"
	AIInputFieldTypeDateTime AIInputType = "DATETIME"
	AIInputFieldTypeBool     AIInputType = "BOOL"
)

type AIWorkerReturnType string

const (
	AIWorkerReturnTypeText AIWorkerReturnType = "TEXT"
	AIWorkerReturnTypeData AIWorkerReturnType = "DATA"
)

type AIDisplayType string

const (
	AIWorkerDisplayTypeTextbox           AIDisplayType = "TEXTBOX"
	AIWorkerDisplayTypeTextarea          AIDisplayType = "TEXTAREA"
	AIWorkerDisplayTypeReadonlyTable     AIDisplayType = "READONLY_TABLE"
	AIWorkerDisplayTypeSingleChoiceTable AIDisplayType = "SINGLE_CHOICE_TABLE"
	AIWorkerDisplayTypeMultiChoiceTable  AIDisplayType = "MULTI_CHOICE_TABLE"
)
