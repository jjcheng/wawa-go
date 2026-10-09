package service

// general
type dashScopeErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"request_id"`
}

func (dashscopeErrorResponse *dashScopeErrorResponse) Error() string {
	return dashscopeErrorResponse.Message
}

// embed
type dashScopeEmbeddingResponse struct {
	RequestId string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Output    struct {
		Embeddings []dashScopeEmbedding `json:"embeddings"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	}
}

type dashScopeEmbedding struct {
	TextIndex int       `json:"text_index"`
	Embedding []float32 `json:"embedding"`
}

// chat
type DashScopeChatModel string

const (
	DashScopeChatModelQwen38Flash DashScopeChatModel = "qwen3.8-flash"
	DashScopeChatModelQwen38Max   DashScopeChatModel = "qwen3.8-max"
)

type DashScopeChatMessageRole string

const (
	DashScopeChatMessageRoleSystem    DashScopeChatMessageRole = "system"
	DashScopeChatMessageRoleAssistant DashScopeChatMessageRole = "assistant"
	DashScopeChatMessageRoleUser      DashScopeChatMessageRole = "user"
)

type DashScopeMessageContentType string

const (
	DashScopeChatMessageContentTypeText  DashScopeMessageContentType = "text"
	DashScopeChatMessageContentTypeImage DashScopeMessageContentType = "image"
	DashScopeChatMessageContentTypeVideo DashScopeMessageContentType = "video"
)

type DashScopeMessage struct {
	Role             DashScopeChatMessageRole `json:"role"`
	Content          any                      `json:"content"`           // string or array [{"image": "https://www.png"}, {"text": "dasdsa"}]
	ReasoningContent string                   `json:"reasoning_content"` // only from response
}

type DashScopeMessageContent struct {
	Type    DashScopeMessageContentType `json:"type"`
	Content string                      `json:"content"`
}

func (dashScopeMessageContent DashScopeMessageContent) payload() map[string]any {
	var m map[string]any = map[string]any{}
	switch dashScopeMessageContent.Type {
	case DashScopeChatMessageContentTypeText:
		m["text"] = dashScopeMessageContent.Content
	case DashScopeChatMessageContentTypeImage:
		m["image"] = dashScopeMessageContent.Content
	}
	return m
}

func (dashScopeMessage DashScopeMessage) Payload() map[string]any {
	o := map[string]any{
		"role": dashScopeMessage.Role,
	}
	if str, ok := dashScopeMessage.Content.(string); ok {
		o["content"] = str
	} else if contents, ok := dashScopeMessage.Content.([]DashScopeMessageContent); ok {
		contentList := []map[string]any{}
		for _, content := range contents {
			contentList = append(contentList, content.payload())
		}
		o["content"] = contentList
	}
	return o
}

type DashScopeQwenThiningEffort string

const (
	DashScopeQwenThinkingEffortXHigh  DashScopeQwenThiningEffort = "xhigh"
	DashScopeQwenThinkingEffortMedium DashScopeQwenThiningEffort = "medium"
	DashScopeQwenThinkingEffortLow    DashScopeQwenThiningEffort = "low"
)

type DashScopeResponseFormat string

const (
	DashScopeResponseFormatText       DashScopeResponseFormat = "text"
	DashScopeResponseFormatJSONObject DashScopeResponseFormat = "json_object"
	DashScopeResponseFormatJSONSchema DashScopeResponseFormat = "json_schema"
)

type DashScopeJSONSchema struct {
	Name   string                    `json:"name"`
	Strict bool                      `json:"strict"`
	Schema DashScopeJSONSchameObject `json:"schema"`
}

type DashScopeJSONSchameObject struct {
	Type                 string                                 `json:"type"`       // always object
	Title                string                                 `json:"title"`      // e.g. UserInfo
	Properties           map[string]DashScopeJSONSchemaProperty `json:"properties"` // ["name": {"title": string,}]
	Required             []string                               `json:"required"`   // ["name"]
	AdditionalProperties bool                                   `json:"additionalProperties"`
}

type DashScopeJSONSchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type DashScopeChatResponse struct {
	StatusCode string                      `json:"status_code"`
	RequestId  string                      `json:"request_id"`
	Code       string                      `json:"code"`
	Message    string                      `json:"message"`
	Output     DashScopeChatResponseOutput `json:"output"`
	Usage      DashScopeChatResponseUsage  `json:"usage"`
}

type DashScopeChatResponseOutput struct {
	Choices []DashScopeChatResponseOutputChoice `json:"choices"`
}

type DashScopeChatResponseOutputChoice struct {
	// empty if generating
	// stop if the model output ends naturally or triggers a stop condition in the input parameters
	// The generation terminates for the reason length, which means the output is too long.
	// The tool_calls reason indicates that a tool call occurred.
	FinishReason string           `json:"finish_reason"`
	Message      DashScopeMessage `json:"message"`
}

type DashScopeChatResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// decide
type DashScopeDecideQuestionType string

const (
	DashScopeDecideQuestionTypeChoice DashScopeDecideQuestionType = "choice"
	DashScopeDecideQuestionTypeNoul   DashScopeDecideQuestionType = "noul"
	DashScopeDecideQuestionTypeScore  DashScopeDecideQuestionType = "score"
)

type DashScopeDecideQuestion map[string]DashScopeDecideQuestionContent

type DashScopeDecideQuestionContent struct {
	Type         DashScopeDecideQuestionType `json:"type"`
	Instructions string                      `json:"instructions"`
	// can be DashScopeDecideQuestionCriteriaChoice, DashScopeDecideQuestionCriteriaNoul or DashScopeDecideQuestionCriteriaScore
	Criteria any `json:"criteria"`
}

type DashScopeDecideQuestionCriteriaChoice map[string]any // {"billing": "Payment, refund and billing issues"}
type DashScopeDecideQuestionCriteriaNoul struct {
	True  string `json:"true"`
	False string `json:"false"`
}
type DashScopeDecideQuestionCriteriaScore []string // ["Minor issue", "Medium issue", "Urgent issue"]

func (dashScopeDecideQuestion DashScopeDecideQuestionContent) Payload() map[string]any {
	var m map[string]any = map[string]any{
		"type":         dashScopeDecideQuestion.Type,
		"instructions": dashScopeDecideQuestion.Instructions,
	}
	if dashScopeDecideQuestion.Type == DashScopeDecideQuestionTypeChoice {
		if c, ok := dashScopeDecideQuestion.Criteria.(map[string]any); ok {
			m["criteria"] = c
		}
	} else if dashScopeDecideQuestion.Type == DashScopeDecideQuestionTypeNoul {
		if c, ok := dashScopeDecideQuestion.Criteria.(DashScopeDecideQuestionCriteriaNoul); ok {
			m["criteria"] = map[string]any{
				"true":  c.True,
				"false": c.False,
			}
		}
	} else {
		if c, ok := dashScopeDecideQuestion.Criteria.([]string); ok {
			m["criteria"] = c
		}
	}
	return m
}

// decide response
type DashScopeDecideResponseAnswers map[string]any

type DashScopeDecideResponseAnswerChoice struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float32            `json:"confidence"`
	Probabilities map[string]float32 `json:"probabilities"`
}

type DashScopeDecideResponseAnswerNoul struct {
	Type string  `json:"type"`
	Noul float32 `json:"noul"`
}

type DashScopeDecideResponseAnswerScore struct {
	Type          string             `json:"type"`
	Score         float32            `json:"score"`
	Confidence    float32            `json:"confidence"`
	Legend        map[string]string  `json:"legend"`
	Probabilities map[string]float32 `json:"probabilities"`
}

type DashScopeDecideResponseUsage struct {
	InputTokens int `json:"input_tokens"`
}

type DashScopeDecideResponse struct {
	Model     string                         `json:"model"`
	RequestId string                         `json:"request_id"`
	Answers   DashScopeDecideResponseAnswers `json:"answers"`
	Usage     DashScopeDecideResponseUsage   `json:"usage"`
	LatencyMS float32                        `json:"latency_ms"`
}
