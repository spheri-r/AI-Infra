package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-inferra/internal/models"
)

type AnthropicProvider struct {
	httpClient *http.Client
	baseURL    string
	apiVersion string
}

func NewAnthropicProvider(baseURL, apiVersion string) *AnthropicProvider {
	return &AnthropicProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		apiVersion: apiVersion,
	}
}

func (ap *AnthropicProvider) ValidateRequest(req *models.ChatCompletionRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages are required")
	}

	// Validate message roles for Anthropic
	for i, msg := range req.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("message %d: role must be 'user' or 'assistant' for Anthropic", i)
		}
		if msg.Content == "" {
			return fmt.Errorf("message %d: content cannot be empty", i)
		}
	}

	// Ensure first message is from user
	if req.Messages[0].Role != "user" {
		return fmt.Errorf("first message must be from user for Anthropic")
	}

	return nil
}

// 函数的核心功能是将标准化的聊天完成请求格式转换为 Anthropic API 特定的请求格式
// 使用 map[string]interface{} 动态类型映射来构建请求，便于序列化为 JSON ；然后进行可选参数条件性添加；还做了openai的兼容性处理；最后返回转换后的标准响应结构。
// 系统对外提供统一的 ChatCompletionRequest 接口，内部通过适配器模式支持多个不同的 LLM 提供商（OpenAI、Anthropic 等），客户端无需关心底层提供商的具体 API 格式
// 这个请求后面会被序列化为json，因此初始构建了一个动态类型的map，后续条件性添加字段，只有当字段存在时才会被添加
func (ap *AnthropicProvider) TransformRequest(req *models.ChatCompletionRequest) (interface{}, error) {
	anthropicReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}

	// Set max_tokens (required for Anthropic)
	if req.MaxTokens != nil {
		anthropicReq["max_tokens"] = *req.MaxTokens
	} else {
		anthropicReq["max_tokens"] = 4096 // Default value
	}

	// Optional parameters
	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}

	if req.TopP != nil {
		anthropicReq["top_p"] = *req.TopP
	}

	if req.System != "" {
		anthropicReq["system"] = req.System
	}

	if req.Stop != nil {
		anthropicReq["stop_sequences"] = req.Stop
	}

	if req.Stream {
		anthropicReq["stream"] = true
	}

	if req.Metadata != nil {
		anthropicReq["metadata"] = req.Metadata
	}

	return anthropicReq, nil
}

// 函数的核心功能是将 Anthropic API 的原生响应格式转换为系统统一的标准响应格式。
// 使用Go的类型断言（type assertion）将通用接口 interface{} 转换为具体的 *models.AnthropicResponse 类型 ；然后进行基础字段映射；还做了openai的兼容性处理；最后返回转换后的标准响应结构。
// 意图是保持对不同供应商（OpenAI、Anthropic等）的响应兼容处理，确保LLM-inferra 可以支持不同的LLM供应商，并保持一个统一的调用接口。
func (ap *AnthropicProvider) TransformResponse(resp interface{}) (*models.ChatCompletionResponse, error) {
	//类型断言语法，将resp断言为AnthropicResponse类型  value, ok := interface{}.(SpecificType)
	anthropicResp, ok := resp.(*models.AnthropicResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	// Transform Anthropic response to standard format
	response := &models.ChatCompletionResponse{
		ID:    anthropicResp.ID,
		Type:  anthropicResp.Type,
		Role:  anthropicResp.Role,
		Model: anthropicResp.Model,
		Usage: models.ChatCompletionUsage{
			InputTokens:      anthropicResp.Usage.InputTokens,
			OutputTokens:     anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			PromptTokens:     anthropicResp.Usage.InputTokens,  // For OpenAI compatibility
			CompletionTokens: anthropicResp.Usage.OutputTokens, // For OpenAI compatibility
		},
	}

	// Transform content
	for _, content := range anthropicResp.Content {
		response.Content = append(response.Content, models.ChatCompletionContent(content))
	}

	// For OpenAI compatibility - create choices array
	if len(anthropicResp.Content) > 0 {
		choice := models.ChatCompletionChoice{
			Index: 0,
			Message: models.ChatMessage{
				Role:    anthropicResp.Role,
				Content: anthropicResp.Content[0].Text,
			},
			FinishReason: anthropicResp.StopReason,
		}
		response.Choices = []models.ChatCompletionChoice{choice}
		response.Object = "chat.completion"
		response.Created = time.Now().Unix()
	}

	return response, nil
}

func (ap *AnthropicProvider) ChatCompletion(ctx *models.LLMRequestContext, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Validate request
	if err := ap.ValidateRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Transform request for Anthropic API
	anthropicReq, err := ap.TransformRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request transformation failed: %w", err)
	}

	// Serialize request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", ap.baseURL+"/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ctx.APIKey.KeyValue)
	httpReq.Header.Set("anthropic-version", ap.apiVersion)

	if req.AnthropicVersion != "" {
		httpReq.Header.Set("anthropic-version", req.AnthropicVersion)
	}

	// Make the request
	httpResp, err := ap.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-2xx status codes
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse Anthropic response
	var anthropicResp models.AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Transform to standard response format
	response, err := ap.TransformResponse(&anthropicResp)
	if err != nil {
		return nil, fmt.Errorf("response transformation failed: %w", err)
	}

	return response, nil
}

// StreamResponse wraps the streaming data with usage information
type StreamResponse struct {
	DataChan  <-chan []byte
	UsageChan <-chan *models.ChatCompletionUsage
	ErrorChan <-chan error
}

func (ap *AnthropicProvider) StreamChatCompletion(ctx *models.LLMRequestContext, req *models.ChatCompletionRequest) (<-chan []byte, error) {
	// Set streaming flag
	req.Stream = true

	// Validate request
	if err := ap.ValidateRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Transform request for Anthropic API
	anthropicReq, err := ap.TransformRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request transformation failed: %w", err)
	}

	// Serialize request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", ap.baseURL+"/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers for streaming
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ctx.APIKey.KeyValue)
	httpReq.Header.Set("anthropic-version", ap.apiVersion)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")

	if req.AnthropicVersion != "" {
		httpReq.Header.Set("anthropic-version", req.AnthropicVersion)
	}

	// Make the request
	httpResp, err := ap.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Handle non-2xx status codes
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Create channel for streaming response
	streamChan := make(chan []byte, 100)

	go func() {
		defer close(streamChan)
		defer httpResp.Body.Close()

		buffer := make([]byte, 4096)
		remainder := ""

		for {
			n, err := httpResp.Body.Read(buffer)
			if n > 0 {
				// 1. 拼接数据：新读取的数据 + 上次的残留数据
				text := remainder + string(buffer[:n])

				// 2. 解析事件：将文本按双换行符分割为多个事件
				events := ap.parseSSEEvents(text)

				// 3. 处理每个事件：
				for i, event := range events {
					if i == len(events)-1 && !strings.HasSuffix(text, "\n\n") {
						// 最后一个事件不完整，保存到 remainder
						remainder = event
						break
					}

					// 4. 处理事件：处理每个事件，提取usage信息，并返回处理后的数据
					if processedData := ap.processSSEEvent(event); processedData != nil {
						streamChan <- processedData
					}
					remainder = "" //清空remainder
				}
			}
			if err != nil {
				if err != io.EOF {
					// Send error as last message
					errorMsg := fmt.Sprintf("data: {\"error\": \"%v\"}\n\n", err)
					streamChan <- []byte(errorMsg)
				}
				break
			}
		}
	}()

	return streamChan, nil
}

// parseSSEEvents parses Server-Sent Events from text data
func (ap *AnthropicProvider) parseSSEEvents(text string) []string {
	// Split by double newlines to separate events
	events := strings.Split(text, "\n\n")
	return events
}

// AnthropicStreamEvent represents different types of streaming events
type AnthropicStreamEvent struct {
	Type  string          `json:"type"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Message *struct {
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
}

// processSSEEvent processes individual SSE events and extracts usage information
func (ap *AnthropicProvider) processSSEEvent(eventText string) []byte {
	lines := strings.Split(strings.TrimSpace(eventText), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Skip empty data or [DONE] markers
			if data == "" || data == "[DONE]" {
				continue
			}

			// Try to parse as Anthropic stream event
			var streamEvent AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &streamEvent); err == nil {
				// Check for usage information in different event types
				if streamEvent.Type == "message_stop" && streamEvent.Message != nil && streamEvent.Message.Usage != nil {
					// Extract usage information from final event
					usage := streamEvent.Message.Usage
					usageEvent := map[string]interface{}{
						"type": "usage_update",
						"usage": map[string]int{
							"input_tokens":  usage.InputTokens,
							"output_tokens": usage.OutputTokens,
							"total_tokens":  usage.InputTokens + usage.OutputTokens,
						},
					}

					// Convert back to SSE format
					usageData, _ := json.Marshal(usageEvent)
					return []byte(fmt.Sprintf("data: %s\n\n", string(usageData)))
				}
			}

			// Return original data for other events (content deltas, etc.)
			return []byte(fmt.Sprintf("data: %s\n\n", data))
		}
	}

	return nil
}

// Helper function to calculate cost
func (ap *AnthropicProvider) CalculateCost(usage *models.ChatCompletionUsage, model *models.LLMModel) (inputCost, outputCost, totalCost float64) {
	// Calculate costs based on token usage and model pricing
	inputCost = float64(usage.InputTokens) * model.InputCostPer1K / 1000.0
	outputCost = float64(usage.OutputTokens) * model.OutputCostPer1K / 1000.0
	totalCost = inputCost + outputCost
	return
}
