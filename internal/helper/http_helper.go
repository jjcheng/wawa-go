package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HTTPRequestError struct {
	StatusCode int
	Message    string
}

func (httpRequestError *HTTPRequestError) Error() string {
	return httpRequestError.Message
}

func RequestHTTP(ctx context.Context, url string, method string, headers *map[string]string, jsonObject *map[string]any) (int, *string, *http.Header, error) {
	var body io.Reader
	if jsonObject != nil {
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonObject); err != nil {
			return http.StatusInternalServerError, nil, nil, &HTTPRequestError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("failed to marshal request JSON: %s", err.Error()),
			}
		}
		body = buffer
	}
	// Create request with context
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return http.StatusInternalServerError, nil, nil, &HTTPRequestError{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("failed to create http request: %s", err.Error()),
		}
	}
	request.Header.Add("accept", "application/json")
	request.Header.Add("content-type", "application/json")
	if headers != nil {
		for key, value := range *headers {
			request.Header.Add(key, value)
		}
	}
	// Use context-aware client
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		// Check if context was cancelled or timed out
		if ctx.Err() != nil {
			msg, status := getStatusCodeFromContext(ctx)
			return status, nil, nil, &HTTPRequestError{
				StatusCode: status,
				Message:    msg,
			}
		}
		return http.StatusInternalServerError, nil, nil, &HTTPRequestError{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("http request failed: %s", err.Error()),
		}
	}
	if response != nil {
		defer response.Body.Close()
	}
	if response == nil {
		return http.StatusInternalServerError, nil, nil, &HTTPRequestError{
			StatusCode: http.StatusInternalServerError,
			Message:    "http response is null",
		}
	}
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, &response.Header, &HTTPRequestError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("failed to read http response body: %s", err.Error()),
		}
	}
	var responseString string
	if responseBytes != nil {
		responseString = string(responseBytes)
	} else {
		return response.StatusCode, nil, &response.Header, &HTTPRequestError{
			StatusCode: response.StatusCode,
			Message:    "http response bytes is null",
		}
	}
	// check for http 429 too many request
	if response.StatusCode == http.StatusTooManyRequests {
		return response.StatusCode, &responseString, &response.Header, &HTTPRequestError{
			StatusCode: response.StatusCode,
			Message:    "you have hit a rate limit",
		}
	}
	return response.StatusCode, &responseString, &response.Header, nil
}

func getStatusCodeFromContext(ctx context.Context) (string, int) {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return "Your request has timed out", http.StatusRequestTimeout // 408
	case context.Canceled:
		return "You have cancelled your request", http.StatusRequestTimeout // 408 or 499 (Client Closed Request)
	default:
		return "We have encountered a server error", http.StatusInternalServerError // 500
	}
}

func GetUserAgent(ctx context.Context) *string {
	if ginCtx := GetGinContext(ctx); ginCtx != nil {
		userAgent := ginCtx.GetHeader("User-Agent")
		if userAgent != "" {
			return &userAgent
		}
	}
	return nil
}

func GetClientIP(ctx context.Context) *string {
	if ginCtx := GetGinContext(ctx); ginCtx != nil {
		clientIP := ginCtx.ClientIP()
		if clientIP != "" {
			return &clientIP
		}
	}
	return nil
}

func GetHostName(ctx context.Context) *string {
	if ginCtx := GetGinContext(ctx); ginCtx != nil {
		host := ginCtx.Request.Host
		if host != "" {
			return &host
		}
	}
	return nil
}

func GetGinContext(ctx context.Context) *gin.Context {
	if ginCtxValue := ctx.Value("gin"); ginCtxValue != nil {
		if ginCtx, ok := ginCtxValue.(*gin.Context); ok {
			return ginCtx
		}
	}
	return nil
}
