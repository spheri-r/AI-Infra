package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Username string `json:"username" gorm:"uniqueIndex;not null" validate:"required,min=3,max=50"`
	Email    string `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Password string `json:"-" gorm:"not null" validate:"required,min=6"`

	FirstName string     `json:"first_name" validate:"required,max=100"`
	LastName  string     `json:"last_name" validate:"required,max=100"`
	Role      UserRole   `json:"role" gorm:"default:user"`
	Status    UserStatus `json:"status" gorm:"default:active"`

	// Usage tracking
	TotalRequests   int64      `json:"total_requests" gorm:"default:0"`
	TotalCost       float64    `json:"total_cost" gorm:"default:0"`
	MonthlyRequests int64      `json:"monthly_requests" gorm:"default:0"`
	MonthlyCost     float64    `json:"monthly_cost" gorm:"default:0"`
	LastRequestAt   *time.Time `json:"last_request_at"`

	// Limits
	DailyRequestLimit   int64   `json:"daily_request_limit" gorm:"default:1000"`
	MonthlyRequestLimit int64   `json:"monthly_request_limit" gorm:"default:10000"`
	DailyCostLimit      float64 `json:"daily_cost_limit" gorm:"default:10.0"`
	MonthlyCostLimit    float64 `json:"monthly_cost_limit" gorm:"default:100.0"`

	// Relationships
	APIKeys   []APIKey   `json:"api_keys,omitempty" gorm:"foreignKey:UserID"`
	UsageLogs []UsageLog `json:"usage_logs,omitempty" gorm:"foreignKey:UserID"`
}

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusInactive  UserStatus = "inactive"
	StatusSuspended UserStatus = "suspended"
)

type CreateUserRequest struct {
	Username  string   `json:"username" validate:"required,min=3,max=50"`
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required,min=6"`
	FirstName string   `json:"first_name" validate:"required,max=100"`
	LastName  string   `json:"last_name" validate:"required,max=100"`
	Role      UserRole `json:"role,omitempty"`
}

type UpdateUserRequest struct {
	FirstName           *string     `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName            *string     `json:"last_name,omitempty" validate:"omitempty,max=100"`
	Role                *UserRole   `json:"role,omitempty"`
	Status              *UserStatus `json:"status,omitempty"`
	DailyRequestLimit   *int64      `json:"daily_request_limit,omitempty"`
	MonthlyRequestLimit *int64      `json:"monthly_request_limit,omitempty"`
	DailyCostLimit      *float64    `json:"daily_cost_limit,omitempty"`
	MonthlyCostLimit    *float64    `json:"monthly_cost_limit,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
