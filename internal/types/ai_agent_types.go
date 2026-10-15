package types

type AIAgentFunctionParameterType string

const (
	AIAgentFunctionParameterTypeString      AIAgentFunctionParameterType = "STRING"
	AIAgentFunctionParameterTypeInt         AIAgentFunctionParameterType = "INT"
	AIAgentFunctionParameterTypeFloat       AIAgentFunctionParameterType = "FLOAT"
	AIAgentFunctionParameterTypeBool        AIAgentFunctionParameterType = "BOOL"
	AIAgentFunctionParameterTypeStringArray AIAgentFunctionParameterType = "STRING_ARRAY"
	AIAgentFunctionParameterTypeIntArray    AIAgentFunctionParameterType = "INT_ARRAY"
	AIAgentFunctionParameterTypeFloatArray  AIAgentFunctionParameterType = "FLOAT_ARRAY"
	AIAgentFunctionParameterTypeBoolArray   AIAgentFunctionParameterType = "BOOL_ARRAY"
)

type AIAgentConnectorAuthType string

const (
	AIAgentConnectorAuthTypeHeaderAPIKey AIAgentConnectorAuthType = "HEADER_API_KEY"
)

type AIAgentConnectorAuthLocation string

const (
	AIAgentConnectorAuthLocationHeader AIAgentConnectorAuthLocation = "HEADER"
)

type AIAgentSkillType string

const (
	AIAgentSkillTypeAnswerQuestion AIAgentSkillType = "ANSWER_QUESTION"
	AIAgentSkillTypeCustom         AIAgentSkillType = "CUSTOM"
)

type AIAgentSkillInputType string

const (
	AIAgentSkillInputTypeText     AIAgentSkillInputType = "TEXT"
	AIAgentSkillInputTypeInt      AIAgentSkillInputType = "INT"
	AIAgentSkillInputTypeFloat    AIAgentSkillInputType = "FLOAT"
	AIAgentSkillInputTypeBool     AIAgentSkillInputType = "BOOL"
	AIAgentSkillInputTypeDate     AIAgentSkillInputType = "DATE"
	AIAgentSkillInputTypeTime     AIAgentSkillInputType = "TIME"
	AIAgentSkillInputTypeDateTime AIAgentSkillInputType = "DATETIME"
)
