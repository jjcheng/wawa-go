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

type AIAgentFAQType string

const (
	AIAgentFAQTypeText     AIAgentFAQType = "TEXT"
	AIAgentFAQTypeImage    AIAgentFAQType = "IMAGE"
	AIAgentFAQTypeVideo    AIAgentFAQType = "VIDEO"
	AIAgentFAQTypeDocument AIAgentFAQType = "DOCUMENT"
	AIAgentFAQTypeLocation AIAgentFAQType = "LOCATION"
)
