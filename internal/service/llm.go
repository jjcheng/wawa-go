package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

type LLM struct {
	apiKey  string
	baseUrl string
	logger  *Logger
}

func NewLLM(logger *Logger) *LLM {
	return &LLM{
		apiKey:  cfg.Default().LLM.APIKey,
		baseUrl: cfg.Default().LLM.BaseURL,
		logger:  logger,
	}
}

func (llm *LLM) Embed(ctx context.Context, textList []string) ([][]float32, int, error) {
	requestBody := map[string]any{
		"model": "qwen3.7-text-embedding",
		"input": map[string]any{
			"texts": textList,
		},
		"parameters": map[string]any{
			"dimension":   1024,
			"output_type": "dense",
		},
	}
	baseUrl := fmt.Sprintf("%s/api/v1/services/embeddings/text-embedding/text-embedding", llm.baseUrl)
	status, response, _, err := helper.RequestHTTP(ctx, baseUrl, "POST", llm.requestHeader(), &requestBody)
	if err != nil {
		return nil, 0, err
	}
	if status != 200 {
		var errorResponse dashScopeErrorResponse
		if err := json.Unmarshal([]byte(*response), &errorResponse); err != nil {
			return nil, 0, fmt.Errorf("failed to get embedding using dashscope with status code: %d", status)
		}
		return nil, 0, fmt.Errorf("failed to get embedding using dashscope: %w", &errorResponse)
	}
	var result dashScopeEmbeddingResponse
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, 0, fmt.Errorf("decode DashScope embedding response: %w", err)
	}
	if len(result.Output.Embeddings) == 0 {
		return nil, 0, fmt.Errorf("DashScope embedding response contains no embedding")
	}
	helper.Sort(result.Output.Embeddings, func(embedding1, embedding2 dashScopeEmbedding) int {
		if embedding1.TextIndex < embedding2.TextIndex {
			return -1
		} else if embedding1.TextIndex > embedding2.TextIndex {
			return 1
		} else {
			return 0
		}
	})
	embeddings := make([][]float32, 0, len(result.Output.Embeddings))
	for i := range result.Output.Embeddings {
		embeddings[i] = result.Output.Embeddings[i].Embedding
	}
	return embeddings, result.Usage.TotalTokens, nil
}

func (llm *LLM) Chat(ctx context.Context, model DashScopeChatModel, messages []DashScopeMessage, temperature float32, thinkingEffort *DashScopeQwenThiningEffort, maxCompletionTokens int, responseFormat DashScopeResponseFormat) (*DashScopeChatResponse, error) {
	parameters := map[string]any{
		"temperature":   temperature,
		"result_format": "message",
	}
	if thinkingEffort != nil {
		parameters["enable_thinking"] = true
		parameters["preserve_thinking"] = true
		parameters["reasoning_effort"] = *thinkingEffort
	} else {
		parameters["enable_thinking"] = false
	}
	if maxCompletionTokens > 0 {
		parameters["max_completion_tokens"] = maxCompletionTokens
	}
	if responseFormat == DashScopeResponseFormatJSONObject {
		parameters["response_format"] = map[string]any{
			"type": "json_object",
		}
	}
	requestBody := map[string]any{
		"model": model,
		"input": map[string]any{
			"messages": helper.Map(messages, func(m DashScopeMessage) map[string]any {
				return m.Payload()
			}),
		},
		"parameters": parameters,
	}
	baseUrl := fmt.Sprintf("%s/api/v1/services/aigc/multimodal-generation/generation", llm.baseUrl)
	// TODO: save response
	status, response, _, err := helper.RequestHTTP(ctx, baseUrl, "POST", llm.requestHeader(), &requestBody)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		var errorResponse dashScopeErrorResponse
		if err := json.Unmarshal([]byte(*response), &errorResponse); err != nil {
			return nil, fmt.Errorf("failed to call chat with status code: %d", status)
		}
		return nil, fmt.Errorf("failed to call chat: %w", &errorResponse)
	}
	var result DashScopeChatResponse
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, fmt.Errorf("decode DashScope chat response: %w", err)
	}
	return &result, nil
}

func (llm *LLM) Decide(ctx context.Context, messages []DashScopeMessage, questions DashScopeDecideQuestion) (*DashScopeDecideResponse, error) {
	var states []map[string]any = helper.Map(messages, func(m DashScopeMessage) map[string]any {
		return m.Payload()
	})
	requestBody := map[string]any{
		"model":     "decision-model-preview",
		"state":     states,
		"questions": questions,
	}
	baseUrl := fmt.Sprintf("%s/compatible-mode/v1/systemone", llm.baseUrl)
	status, response, _, err := helper.RequestHTTP(ctx, baseUrl, "POST", llm.requestHeader(), &requestBody)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		var errorResponse dashScopeErrorResponse
		if err := json.Unmarshal([]byte(*response), &errorResponse); err != nil {
			return nil, fmt.Errorf("failed to call chat with status code: %d", status)
		}
		return nil, fmt.Errorf("failed to call chat: %w", &errorResponse)
	}
	var result DashScopeDecideResponse
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, fmt.Errorf("decode DashScope chat response: %w", err)
	}
	return &result, nil
}

func (llm *LLM) requestHeader() *map[string]string {
	return &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", llm.apiKey),
	}
}
