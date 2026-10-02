package service

import (
	"context"
	"fmt"

	bailian "github.com/aliyun/alibabacloud-bailian-go-sdk/client"
)

type LLM struct {
	logger     *Logger
	qwenClient *bailian.CompletionClient
	appID      string
}

func NewLLM(logger *Logger) *LLM {
	//config := cfg.Default().LLM
	return nil
	// if strings.TrimSpace(config.AccessKeyID) == "" || strings.TrimSpace(config.AccessKeySecret) == "" {
	// 	panic("missing Alibaba Cloud credentials when creating LLM service")
	// }
	// if strings.TrimSpace(config.AgentKey) == "" || strings.TrimSpace(config.AppID) == "" {
	// 	panic("missing DashScope agent key or app ID when creating LLM service")
	// }

	// tokenClient := bailian.AccessTokenClient{
	// 	AccessKeyId:     strings.TrimSpace(config.AccessKeyID),
	// 	AccessKeySecret: strings.TrimSpace(config.AccessKeySecret),
	// 	AgentKey:        strings.TrimSpace(config.AgentKey),
	// 	Endpoint:        strings.TrimSpace(config.Endpoint),
	// }
	// token, err := tokenClient.GetToken()
	// if err != nil {
	// 	panic(fmt.Errorf("failed to get DashScope access token: %w", err))
	// }

	// completionClient := &bailian.CompletionClient{
	// 	Token:    token,
	// 	Endpoint: strings.TrimSpace(config.Endpoint),
	// 	Timeout:  60 * time.Second,
	// }
	// return &LLM{logger: logger, qwenClient: completionClient, appID: strings.TrimSpace(config.AppID)}
}

func (llm *LLM) Complete(ctx context.Context, messages []bailian.ChatCompletionMessage) (string, error) {
	_ = ctx
	if len(messages) == 0 {
		return "", fmt.Errorf("LLM messages must not be empty")
	}
	if llm == nil || llm.qwenClient == nil {
		return "", fmt.Errorf("LLM client is not initialized")
	}

	request := &bailian.CompletionRequest{
		AppId:    llm.appID,
		Messages: messages,
	}
	response, err := llm.qwenClient.CreateCompletion(request)
	if err != nil {
		return "", fmt.Errorf("call DashScope: %w", err)
	}
	if response == nil || !response.Success || response.Data == nil {
		if response == nil {
			return "", fmt.Errorf("DashScope returned an empty response")
		}
		return "", fmt.Errorf("DashScope completion failed: code=%s message=%s", response.Code, response.Message)
	}
	return response.Data.Text, nil
}
