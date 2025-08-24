package database

import (
	"fmt"
	"log"
	"time"

	"llm-inferra/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabasePoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

func Initialize(databaseURL string, poolConfig DatabasePoolConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for better performance and resource management
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)

	log.Println("Database connection established with connection pool configured")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Provider{},
		&models.LLMModel{},
		&models.APIKey{},
		&models.UsageLog{},
		&models.RequestLog{},
		&models.SystemHealth{},
		&models.LLMRequestLog{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default admin user if it doesn't exist
	if err := seedDefaultUser(db); err != nil {
		return fmt.Errorf("failed to seed default user: %w", err)
	}

	// Create default providers if they don't exist
	if err := seedDefaultProviders(db); err != nil {
		return fmt.Errorf("failed to seed default providers: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func seedDefaultUser(db *gorm.DB) error {
	// Check if admin user already exists
	var existingUser models.User
	if err := db.Where("username = ?", "admin").First(&existingUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Hash the default password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			// Create default admin user
			adminUser := models.User{
				Username:            "admin",
				Email:               "admin@llminferra.com",
				FirstName:           "Admin",
				LastName:            "User",
				Password:            string(hashedPassword),
				Role:                models.RoleAdmin,
				Status:              models.StatusActive,
				DailyRequestLimit:   10000,
				MonthlyRequestLimit: 100000,
				DailyCostLimit:      100.0,
				MonthlyCostLimit:    1000.0,
			}

			if err := db.Create(&adminUser).Error; err != nil {
				return fmt.Errorf("failed to create admin user: %w", err)
			}

			log.Printf("Created default admin user: username=admin, password=admin123")
		} else {
			return fmt.Errorf("failed to check for existing admin user: %w", err)
		}
	}

	return nil
}

func seedDefaultProviders(db *gorm.DB) error {
	providers := []models.Provider{
		{
			Name:                "OpenAI",
			Type:                models.ProviderOpenAI,
			Description:         "OpenAI Language Models",
			BaseURL:             "https://api.openai.com/v1",
			APIVersion:          "v1",
			DefaultRateLimit:    60,
			DefaultCostPerToken: 0.001,
		},
		{
			Name:                "Anthropic",
			Type:                models.ProviderAnthropic,
			Description:         "Anthropic Claude Models",
			BaseURL:             "https://api.anthropic.com/v1",
			APIVersion:          "2023-06-01",
			DefaultRateLimit:    60,
			DefaultCostPerToken: 0.001,
		},
		{
			Name:                "Google",
			Type:                models.ProviderGoogle,
			Description:         "Google AI Models",
			BaseURL:             "https://generativelanguage.googleapis.com/v1",
			APIVersion:          "v1",
			DefaultRateLimit:    60,
			DefaultCostPerToken: 0.001,
		},
	}

	for _, provider := range providers {
		var existingProvider models.Provider
		if err := db.Where("name = ?", provider.Name).First(&existingProvider).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&provider).Error; err != nil {
					return fmt.Errorf("failed to create provider %s: %w", provider.Name, err)
				}
				log.Printf("Created default provider: %s", provider.Name)
			} else {
				return fmt.Errorf("failed to check for existing provider %s: %w", provider.Name, err)
			}
		}
	}

	// Create default models for OpenAI
	return seedDefaultModels(db)
}

func seedDefaultModels(db *gorm.DB) error {
	// Get OpenAI provider
	var openaiProvider models.Provider
	if err := db.Where("name = ?", "OpenAI").First(&openaiProvider).Error; err != nil {
		return fmt.Errorf("failed to find OpenAI provider: %w", err)
	}

	// Get Anthropic provider
	var anthropicProvider models.Provider
	if err := db.Where("name = ?", "Anthropic").First(&anthropicProvider).Error; err != nil {
		return fmt.Errorf("failed to find Anthropic provider: %w", err)
	}

	defaultModels := []models.LLMModel{
		{
			ProviderID:        openaiProvider.ID,
			Name:              "GPT-4 Turbo",
			ModelID:           "gpt-4-turbo-preview",
			Description:       "Most capable GPT-4 model with 128k context",
			MaxTokens:         128000,
			InputCostPer1K:    0.01,
			OutputCostPer1K:   0.03,
			SupportsStreaming: true,
			SupportsFunctions: true,
			SupportsVision:    true,
		},
		{
			ProviderID:        openaiProvider.ID,
			Name:              "GPT-3.5 Turbo",
			ModelID:           "gpt-3.5-turbo",
			Description:       "Fast and efficient GPT-3.5 model",
			MaxTokens:         4096,
			InputCostPer1K:    0.001,
			OutputCostPer1K:   0.002,
			SupportsStreaming: true,
			SupportsFunctions: true,
		},
		{
			ProviderID:        anthropicProvider.ID,
			Name:              "Claude 3 Sonnet",
			ModelID:           "claude-3-sonnet-20240229",
			Description:       "Balanced Claude model for various tasks",
			MaxTokens:         200000,
			InputCostPer1K:    0.003,
			OutputCostPer1K:   0.015,
			SupportsStreaming: true,
		},
		{
			ProviderID:        anthropicProvider.ID,
			Name:              "Claude 3 Haiku",
			ModelID:           "claude-3-haiku-20240307",
			Description:       "Fast and efficient Claude model",
			MaxTokens:         200000,
			InputCostPer1K:    0.00025,
			OutputCostPer1K:   0.00125,
			SupportsStreaming: true,
		},
	}

	for _, model := range defaultModels {
		var existingModel models.LLMModel
		if err := db.Where("model_id = ? AND provider_id = ?", model.ModelID, model.ProviderID).First(&existingModel).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&model).Error; err != nil {
					return fmt.Errorf("failed to create model %s: %w", model.Name, err)
				}
				log.Printf("Created default model: %s", model.Name)
			} else {
				return fmt.Errorf("failed to check for existing model %s: %w", model.Name, err)
			}
		}
	}

	return nil
}
