package types

type AIKnowledgeType string

const (
	AIKnowledgeTypeText     AIKnowledgeType = "TEXT"
	AIKnowledgeTypeImage    AIKnowledgeType = "IMAGE"
	AIKnowledgeTypeDocument AIKnowledgeType = "DOCUMENT"
	AIKnowledgeTypeVideo    AIKnowledgeType = "VIDEO"
)

type AIKnowledgeCategory string

const (
	AIKnowledgeCategoryFAQ     AIKnowledgeCategory = "FAQ"
	AIKnowledgeCategoryFile    AIKnowledgeCategory = "FILE"
	AIKnowledgeCategoryWebsite AIKnowledgeCategory = "WEBSITE"
)

const AIErrorResponse string = "I am sorry, but I could not answer your query."

type AIChatNextStepType string

const (
	AIChatNextStepTypeNotifyUser AIChatNextStepType = "NOTIFY_USER"
)
