package models

import (
	"time"

	"gorm.io/gorm"
)

type UsageLog struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID   uint     `json:"user_id" gorm:"not null"`
	User     User     `json:"user,omitempty"`
	APIKeyID uint     `json:"api_key_id" gorm:"not null"`
	APIKey   APIKey   `json:"api_key,omitempty"`
	ModelID  uint     `json:"model_id" gorm:"not null"`
	Model    LLMModel `json:"model,omitempty"`

	// Request details
	RequestID  string     `json:"request_id" gorm:"uniqueIndex;not null"`
	Endpoint   string     `json:"endpoint" gorm:"not null"`
	Method     string     `json:"method" gorm:"not null"`
	RequestAt  time.Time  `json:"request_at" gorm:"not null"`
	ResponseAt *time.Time `json:"response_at"`

	// Token usage
	InputTokens  int `json:"input_tokens" gorm:"default:0"`
	OutputTokens int `json:"output_tokens" gorm:"default:0"`
	TotalTokens  int `json:"total_tokens" gorm:"default:0"`

	// Cost calculation
	InputCost  float64 `json:"input_cost" gorm:"default:0"`
	OutputCost float64 `json:"output_cost" gorm:"default:0"`
	TotalCost  float64 `json:"total_cost" gorm:"default:0"`

	// Response details
	StatusCode   int    `json:"status_code"`
	Success      bool   `json:"success" gorm:"default:false"`
	ErrorMessage string `json:"error_message"`
	ResponseTime int64  `json:"response_time"` // in milliseconds

	// Additional metadata
	UserAgent    string `json:"user_agent"`
	IPAddress    string `json:"ip_address"`
	RequestSize  int64  `json:"request_size"`  // in bytes
	ResponseSize int64  `json:"response_size"` // in bytes
}

// Analytics structures
type UsageAnalytics struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulRequests  int64   `json:"successful_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	TotalCost           float64 `json:"total_cost"`
	TotalTokens         int64   `json:"total_tokens"`
	AverageResponseTime float64 `json:"average_response_time"`

	// Time-based metrics
	DailyRequests  []DailyMetric  `json:"daily_requests,omitempty"`
	HourlyRequests []HourlyMetric `json:"hourly_requests,omitempty"`

	// Provider metrics
	ProviderMetrics []ProviderMetric `json:"provider_metrics,omitempty"`
	ModelMetrics    []ModelMetric    `json:"model_metrics,omitempty"`
	UserMetrics     []UserMetric     `json:"user_metrics,omitempty"`
}

type DailyMetric struct {
	Date     string  `json:"date"`
	Requests int64   `json:"requests"`
	Cost     float64 `json:"cost"`
	Tokens   int64   `json:"tokens"`
}

type HourlyMetric struct {
	Hour     int     `json:"hour"`
	Requests int64   `json:"requests"`
	Cost     float64 `json:"cost"`
}

type ProviderMetric struct {
	ProviderID   uint    `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Requests     int64   `json:"requests"`
	Cost         float64 `json:"cost"`
	SuccessRate  float64 `json:"success_rate"`
}

type ModelMetric struct {
	ModelID     uint    `json:"model_id"`
	ModelName   string  `json:"model_name"`
	Requests    int64   `json:"requests"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"success_rate"`
}

type UserMetric struct {
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	Requests    int64     `json:"requests"`
	Cost        float64   `json:"cost"`
	LastRequest time.Time `json:"last_request"`
}

// Request/Response logging
type RequestLog struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UsageLogID uint     `json:"usage_log_id" gorm:"not null"`
	UsageLog   UsageLog `json:"usage_log,omitempty"`

	RequestHeaders  string `json:"request_headers" gorm:"type:text"`
	RequestBody     string `json:"request_body" gorm:"type:text"`
	ResponseHeaders string `json:"response_headers" gorm:"type:text"`
	ResponseBody    string `json:"response_body" gorm:"type:text"`
}

// System health and monitoring
type SystemHealth struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`

	// API metrics
	TotalRequests       int64   `json:"total_requests"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
	AverageResponseTime float64 `json:"average_response_time"`
	ErrorRate           float64 `json:"error_rate"`

	// System metrics
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`

	// Database metrics
	DatabaseConnections int `json:"database_connections"`
	DatabaseLatency     int `json:"database_latency"` // in milliseconds

	// Provider status
	ProvidersOnline  int `json:"providers_online"`
	ProvidersOffline int `json:"providers_offline"`
}
