package types

type WATemplateCategory string

const (
	WATemplateCategoryMarketing      WATemplateCategory = "MARKETING"
	WATemplateCategoryUtility        WATemplateCategory = "UTILITY"
	WATemplateCategoryAuthentication WATemplateCategory = "AUTHENTICATION"
)

type WATemplateStatus string

const (
	WATemplateStatusPending  WATemplateStatus = "PENDING"
	WATemplateStatusApproved WATemplateStatus = "APPROVED"
	WATemplateStatusRejected WATemplateStatus = "REJECTED"
	WATemplateStatusInAppeal WATemplateStatus = "IN_APPEAL"
	WATemplateStatusPaused   WATemplateStatus = "PAUSED"
	WATemplateStatusDisabled WATemplateStatus = "DISABLED"
)

type WATemplateQualityScore string

const (
	WATemplateQualityScoreUnknown WATemplateQualityScore = "UNKNOWN"
	WATemplateQualityScoreGreen   WATemplateQualityScore = "GREEN"
	WATemplateQualityScoreYellow  WATemplateQualityScore = "YELLOW"
	WATemplateQualityScoreRed     WATemplateQualityScore = "RED"
)

type WATemplateComponentType string

const (
	WATemplateComponentTypeHeader                WATemplateComponentType = "HEADER"
	WATemplateComponentTypeBody                  WATemplateComponentType = "BODY"
	WATemplateComponentTypeFooter                WATemplateComponentType = "FOOTER"
	WATemplateComponentTypeButtons               WATemplateComponentType = "BUTTONS"
	WATemplateComponentTypeCallPermissionRequest WATemplateComponentType = "CALL_PERMISSION_REQUEST"
)

type WATemplateComponentFormat string

const (
	WATemplateComponentFormatText     WATemplateComponentFormat = "TEXT"
	WATemplateComponentFormatImage    WATemplateComponentFormat = "IMAGE"
	WATemplateComponentFormatVideo    WATemplateComponentFormat = "VIDEO"
	WATemplateComponentFormatDocument WATemplateComponentFormat = "DOCUMENT"
	WATemplateComponentFormatLocation WATemplateComponentFormat = "LOCATION"
)

type WATemplateParameterFormat string

const (
	WATemplateParameterFormatNamed      WATemplateParameterFormat = "NAMED"
	WATemplateParameterFormatPositional WATemplateParameterFormat = "POSITIONAL"
)

type WATemplateButtonType string

const (
	WATemplateButtonTypePhoneNumber WATemplateButtonType = "PHONE_NUMBER"
	WATemplateButtonTypeURL         WATemplateButtonType = "URL"
	WATemplateButtonTypeQuickReply  WATemplateButtonType = "QUICK_REPLY"
	WATemplateButtonTypeVoiceCall   WATemplateButtonType = "VOICE_CALL"
	WATemplateButtonTypeFlow        WATemplateButtonType = "FLOW"
	WATemplateButtonTypeCopyCode    WATemplateButtonType = "COPY_CODE"
)

type WATemplateFlowAction string

const (
	WATemplateFlowActionNavigate WATemplateFlowAction = "NAVIGATE"
)
