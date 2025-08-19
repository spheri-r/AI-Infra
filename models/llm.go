package models

import (
	"encoding/json"
	"time"
)

// LLM Chat Completion Request/Response models
type ChatCompletionRequest struct {
	Model       string                 `json:"model" validate:"required"`
	Messages    []ChatMessage          `json:"messages" validate:"required"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        interface{}            `json:"stop,omitempty"`
	System      string                 `json:"system,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	// Anthropic specific fields
	AnthropicVersion string `json:"anthropic_version,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role" validate:"required"`
	Content string `json:"content" validate:"required"`
}

type ChatCompletionResponse struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type,omitempty"`
	Role    string                  `json:"role,omitempty"`
	Model   string                  `json:"model"`
	Content []ChatCompletionContent `json:"content,omitempty"`
	Usage   ChatCompletionUsage     `json:"usage"`
	// For OpenAI compatibility
	Object  string                 `json:"object,omitempty"`
	Created int64                  `json:"created,omitempty"`
	Choices []ChatCompletionChoice `json:"choices,omitempty"`
}

type ChatCompletionContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	// For OpenAI compatibility
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
}

// Anthropic specific response structures
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Model        string             `json:"model"`
	Content      []AnthropicContent `json:"content"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        AnthropicUsage     `json:"usage"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// LLM Request Log for tracking
type LLMRequestLog struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"index"`

	// Request identification
	RequestID string `json:"request_id" gorm:"uniqueIndex;not null"`
	UserID    uint   `json:"user_id" gorm:"not null"`
	User      User   `json:"user,omitempty"`
	APIKeyID  uint   `json:"api_key_id" gorm:"not null"`
	APIKey    APIKey `json:"api_key,omitempty"`

	// Provider and model info
	ProviderID uint     `json:"provider_id" gorm:"not null"`
	Provider   Provider `json:"provider,omitempty"`
	ModelID    uint     `json:"model_id" gorm:"not null"`
	Model      LLMModel `json:"model,omitempty"`
	ModelName  string   `json:"model_name" gorm:"not null"`

	// Request details
	RequestData  json.RawMessage `json:"request_data" gorm:"type:jsonb"`
	ResponseData json.RawMessage `json:"response_data" gorm:"type:jsonb"`

	// Metrics
	InputTokens  int     `json:"input_tokens" gorm:"default:0"`
	OutputTokens int     `json:"output_tokens" gorm:"default:0"`
	TotalTokens  int     `json:"total_tokens" gorm:"default:0"`
	LatencyMs    int64   `json:"latency_ms" gorm:"default:0"`
	InputCost    float64 `json:"input_cost" gorm:"default:0"`
	OutputCost   float64 `json:"output_cost" gorm:"default:0"`
	TotalCost    float64 `json:"total_cost" gorm:"default:0"`

	// Status
	Status       string `json:"status" gorm:"default:pending"` // pending, completed, failed
	ErrorMessage string `json:"error_message"`
	HTTPStatus   int    `json:"http_status" gorm:"default:0"`

	// Client info
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
}

// Request processing metadata
type LLMRequestContext struct {
	RequestID string
	UserID    uint
	APIKeyID  uint
	Provider  *Provider
	Model     *LLMModel
	APIKey    *APIKey
	ClientIP  string
	UserAgent string
	StartTime time.Time
}

// Provider adapter interface
type LLMProvider interface {
	ChatCompletion(ctx *LLMRequestContext, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	StreamChatCompletion(ctx *LLMRequestContext, req *ChatCompletionRequest) (<-chan []byte, error)
	ValidateRequest(req *ChatCompletionRequest) error
	TransformRequest(req *ChatCompletionRequest) (interface{}, error)
	TransformResponse(resp interface{}) (*ChatCompletionResponse, error)
}
