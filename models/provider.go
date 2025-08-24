package models

import (
	"time"

	"gorm.io/gorm"
)

type Provider struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name        string         `json:"name" gorm:"uniqueIndex;not null" validate:"required"`
	Type        ProviderType   `json:"type" gorm:"not null" validate:"required"`
	Status      ProviderStatus `json:"status" gorm:"default:active"`
	Description string         `json:"description"`

	// Configuration
	BaseURL    string `json:"base_url"`
	APIVersion string `json:"api_version"`

	// Rate limiting and costs
	DefaultRateLimit    int     `json:"default_rate_limit" gorm:"default:60"`        // requests per minute
	DefaultCostPerToken float64 `json:"default_cost_per_token" gorm:"default:0.001"` // per 1K tokens

	// Relationships
	Models  []LLMModel `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
	APIKeys []APIKey   `json:"api_keys,omitempty" gorm:"foreignKey:ProviderID"`
}

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGoogle    ProviderType = "google"
	ProviderCustom    ProviderType = "custom"
)

type ProviderStatus string

const (
	ProviderStatusActive      ProviderStatus = "active"
	ProviderStatusInactive    ProviderStatus = "inactive"
	ProviderStatusMaintenance ProviderStatus = "maintenance"
)

type LLMModel struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	ProviderID uint     `json:"provider_id" gorm:"not null"`
	Provider   Provider `json:"provider,omitempty"`

	Name        string      `json:"name" gorm:"not null" validate:"required"`
	ModelID     string      `json:"model_id" gorm:"not null" validate:"required"` // The actual model identifier for API calls
	Description string      `json:"description"`
	Status      ModelStatus `json:"status" gorm:"default:active"`

	// Model specifications
	MaxTokens       int     `json:"max_tokens" gorm:"default:4096"`
	InputCostPer1K  float64 `json:"input_cost_per_1k" gorm:"default:0.001"`
	OutputCostPer1K float64 `json:"output_cost_per_1k" gorm:"default:0.002"`

	// Model capabilities
	SupportsStreaming  bool `json:"supports_streaming" gorm:"default:true"`
	SupportsFunctions  bool `json:"supports_functions" gorm:"default:false"`
	SupportsVision     bool `json:"supports_vision" gorm:"default:false"`
	SupportsEmbeddings bool `json:"supports_embeddings" gorm:"default:false"`

	// Usage tracking
	TotalRequests int64   `json:"total_requests" gorm:"default:0"`
	TotalCost     float64 `json:"total_cost" gorm:"default:0"`

	// Relationships
	UsageLogs []UsageLog `json:"usage_logs,omitempty" gorm:"foreignKey:ModelID"`
}

type ModelStatus string

const (
	ModelStatusActive     ModelStatus = "active"
	ModelStatusInactive   ModelStatus = "inactive"
	ModelStatusDeprecated ModelStatus = "deprecated"
)

type APIKey struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID     uint     `json:"user_id" gorm:"not null"`
	User       User     `json:"user,omitempty"`
	ProviderID uint     `json:"provider_id" gorm:"not null"`
	Provider   Provider `json:"provider,omitempty"`

	Name      string       `json:"name" gorm:"not null" validate:"required"`
	KeyValue  string       `json:"-" gorm:"not null"` // Encrypted API key
	Status    APIKeyStatus `json:"status" gorm:"default:active"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`

	// Usage limits for this specific API key
	DailyRequestLimit   int64   `json:"daily_request_limit" gorm:"default:1000"`
	MonthlyRequestLimit int64   `json:"monthly_request_limit" gorm:"default:10000"`
	DailyCostLimit      float64 `json:"daily_cost_limit" gorm:"default:10.0"`
	MonthlyCostLimit    float64 `json:"monthly_cost_limit" gorm:"default:100.0"`

	// Usage tracking
	TotalRequests   int64      `json:"total_requests" gorm:"default:0"`
	TotalCost       float64    `json:"total_cost" gorm:"default:0"`
	DailyRequests   int64      `json:"daily_requests" gorm:"default:0"`
	DailyCost       float64    `json:"daily_cost" gorm:"default:0"`
	MonthlyRequests int64      `json:"monthly_requests" gorm:"default:0"`
	MonthlyCost     float64    `json:"monthly_cost" gorm:"default:0"`
	LastUsedAt      *time.Time `json:"last_used_at"`

	// Relationships
	UsageLogs []UsageLog `json:"usage_logs,omitempty" gorm:"foreignKey:APIKeyID"`
}

type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusInactive APIKeyStatus = "inactive"
	APIKeyStatusRevoked  APIKeyStatus = "revoked"
)

type CreateProviderRequest struct {
	Name        string       `json:"name" validate:"required"`
	Type        ProviderType `json:"type" validate:"required"`
	Description string       `json:"description"`
	BaseURL     string       `json:"base_url"`
	APIVersion  string       `json:"api_version"`
}

type CreateModelRequest struct {
	ProviderID         uint    `json:"provider_id" validate:"required"`
	Name               string  `json:"name" validate:"required"`
	ModelID            string  `json:"model_id" validate:"required"`
	Description        string  `json:"description"`
	MaxTokens          int     `json:"max_tokens"`
	InputCostPer1K     float64 `json:"input_cost_per_1k"`
	OutputCostPer1K    float64 `json:"output_cost_per_1k"`
	SupportsStreaming  bool    `json:"supports_streaming"`
	SupportsFunctions  bool    `json:"supports_functions"`
	SupportsVision     bool    `json:"supports_vision"`
	SupportsEmbeddings bool    `json:"supports_embeddings"`
}

type CreateAPIKeyRequest struct {
	ProviderID          uint    `json:"provider_id" validate:"required"`
	Name                string  `json:"name" validate:"required"`
	KeyValue            string  `json:"key_value" validate:"required"`
	DailyRequestLimit   int64   `json:"daily_request_limit"`
	MonthlyRequestLimit int64   `json:"monthly_request_limit"`
	DailyCostLimit      float64 `json:"daily_cost_limit"`
	MonthlyCostLimit    float64 `json:"monthly_cost_limit"`
}
