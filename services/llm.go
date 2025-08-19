package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llm-inferra/internal/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LLMService struct {
	db               *gorm.DB
	redis            *redis.Client
	cache            *CacheService
	providers        map[models.ProviderType]models.LLMProvider
	apiKeyService    *APIKeyService
	providerService  *ProviderService
	analyticsService *AnalyticsService
}

func NewLLMService(db *gorm.DB, redis *redis.Client, apiKeyService *APIKeyService, providerService *ProviderService, analyticsService *AnalyticsService) *LLMService {
	service := &LLMService{
		db:               db,
		redis:            redis,
		cache:            NewCacheService(redis),
		providers:        make(map[models.ProviderType]models.LLMProvider),
		apiKeyService:    apiKeyService,
		providerService:  providerService,
		analyticsService: analyticsService,
	}

	// Initialize providers
	service.initializeProviders()

	return service
}

func (s *LLMService) initializeProviders() {
	// Initialize Anthropic provider
	s.providers[models.ProviderAnthropic] = NewAnthropicProvider(
		"https://api.anthropic.com/v1",
		"2023-06-01",
	)

	// TODO: Add other providers (OpenAI, Google, etc.)
}

// ValidateAPIKey - 为了向后兼容保留的方法，内部调用优化版本
// 推荐直接使用 ValidateAPIKeyOptimized 获得更好的性能
func (s *LLMService) ValidateAPIKey(apiKey string) (*models.LLMRequestContext, error) {
	return s.ValidateAPIKeyOptimized(apiKey)
}

func (s *LLMService) GetModelByName(providerID uint, modelName string) (*models.LLMModel, error) {
	var model models.LLMModel
	err := s.db.Where("provider_id = ? AND model_id = ? AND status = ?", providerID, modelName, models.ModelStatusActive).First(&model).Error
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", modelName)
	}
	return &model, nil
}

func (s *LLMService) ChatCompletion(ctx *models.LLMRequestContext, req *models.ChatCompletionRequest, clientIP, userAgent string) (*models.ChatCompletionResponse, error) {
	// Update context with client info
	ctx.ClientIP = clientIP
	ctx.UserAgent = userAgent

	// Get model information
	model, err := s.GetModelByName(ctx.Provider.ID, req.Model)
	if err != nil {
		return nil, err
	}
	ctx.Model = model

	// Get provider implementation
	provider, exists := s.providers[ctx.Provider.Type]
	if !exists {
		return nil, fmt.Errorf("provider %s not supported", ctx.Provider.Type)
	}

	// Create request log
	requestLog, err := s.createRequestLog(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create request log: %w", err)
	}

	// Make the API call
	startTime := time.Now()
	response, err := provider.ChatCompletion(ctx, req)
	latency := time.Since(startTime)

	// Update request log with response
	if err != nil {
		s.updateRequestLogError(requestLog.ID, err, int(latency.Milliseconds()))
		return nil, err
	}

	// Calculate costs
	var inputCost, outputCost, totalCost float64
	if anthropicProvider, ok := provider.(*AnthropicProvider); ok {
		inputCost, outputCost, totalCost = anthropicProvider.CalculateCost(&response.Usage, model)
	}

	// Update request log with success
	err = s.updateRequestLogSuccess(requestLog.ID, response, &response.Usage, inputCost, outputCost, totalCost, int(latency.Milliseconds()))
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update request log: %v\n", err)
	}

	return response, nil
}

func (s *LLMService) StreamChatCompletion(ctx *models.LLMRequestContext, req *models.ChatCompletionRequest, clientIP, userAgent string) (<-chan []byte, error) {
	// Update context with client info
	ctx.ClientIP = clientIP
	ctx.UserAgent = userAgent

	// Get model information
	model, err := s.GetModelByName(ctx.Provider.ID, req.Model)
	if err != nil {
		return nil, err
	}
	ctx.Model = model

	// Get provider implementation
	provider, exists := s.providers[ctx.Provider.Type]
	if !exists {
		return nil, fmt.Errorf("provider %s not supported", ctx.Provider.Type)
	}

	// Create request log
	requestLog, err := s.createRequestLog(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create request log: %w", err)
	}

	// Make the streaming API call
	startTime := time.Now()
	streamChan, err := provider.StreamChatCompletion(ctx, req)
	if err != nil {
		latency := time.Since(startTime)
		s.updateRequestLogError(requestLog.ID, err, int(latency.Milliseconds()))
		return nil, err
	}

	// Wrap the stream to track completion and extract usage
	wrappedChan := make(chan []byte, 100)

	go func() {
		defer close(wrappedChan)

		var finalUsage *models.ChatCompletionUsage

		for data := range streamChan {
			// Check if this is a usage update event
			if usage := s.extractUsageFromSSE(data); usage != nil {
				finalUsage = usage
				// Don't forward usage events to client, just track internally
				continue
			}

			// Forward regular content events to client
			wrappedChan <- data
		}

		// Update request log with final usage and cost information
		latency := time.Since(startTime)
		if finalUsage != nil {
			// Calculate costs for streaming response
			var inputCost, outputCost, totalCost float64
			if anthropicProvider, ok := provider.(*AnthropicProvider); ok {
				inputCost, outputCost, totalCost = anthropicProvider.CalculateCost(finalUsage, model)
			}

			// Update with complete usage information
			s.updateRequestLogStreamSuccess(requestLog.ID, finalUsage, inputCost, outputCost, totalCost, int(latency.Milliseconds()))
		} else {
			// Fallback: just mark as completed without usage info
			s.updateRequestLogStreamComplete(requestLog.ID, int(latency.Milliseconds()))
		}
	}()

	return wrappedChan, nil
}

func (s *LLMService) createRequestLog(ctx *models.LLMRequestContext, req *models.ChatCompletionRequest) (*models.LLMRequestLog, error) {
	requestData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	log := &models.LLMRequestLog{
		RequestID:   ctx.RequestID,
		UserID:      ctx.UserID,
		APIKeyID:    ctx.APIKeyID,
		ProviderID:  ctx.Provider.ID,
		ModelID:     ctx.Model.ID,
		ModelName:   req.Model,
		RequestData: requestData,
		Status:      "pending",
		ClientIP:    ctx.ClientIP,
		UserAgent:   ctx.UserAgent,
	}

	err = s.db.Create(log).Error
	return log, err
}

func (s *LLMService) updateRequestLogSuccess(logID uint, response *models.ChatCompletionResponse, usage *models.ChatCompletionUsage, inputCost, outputCost, totalCost float64, latencyMs int) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	updates := map[string]interface{}{
		"status":        "completed",
		"response_data": responseData,
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
		"total_tokens":  usage.TotalTokens,
		"input_cost":    inputCost,
		"output_cost":   outputCost,
		"total_cost":    totalCost,
		"latency_ms":    latencyMs,
		"http_status":   200,
	}

	return s.db.Model(&models.LLMRequestLog{}).Where("id = ?", logID).Updates(updates).Error
}

func (s *LLMService) updateRequestLogError(logID uint, err error, latencyMs int) error {
	updates := map[string]interface{}{
		"status":        "failed",
		"error_message": err.Error(),
		"latency_ms":    latencyMs,
		"http_status":   500,
	}

	return s.db.Model(&models.LLMRequestLog{}).Where("id = ?", logID).Updates(updates).Error
}

func (s *LLMService) updateRequestLogStreamComplete(logID uint, latencyMs int) error {
	updates := map[string]interface{}{
		"status":      "completed",
		"latency_ms":  latencyMs,
		"http_status": 200,
	}

	return s.db.Model(&models.LLMRequestLog{}).Where("id = ?", logID).Updates(updates).Error
}

func (s *LLMService) updateRequestLogStreamSuccess(logID uint, usage *models.ChatCompletionUsage, inputCost, outputCost, totalCost float64, latencyMs int) error {
	updates := map[string]interface{}{
		"status":        "completed",
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
		"total_tokens":  usage.TotalTokens,
		"input_cost":    inputCost,
		"output_cost":   outputCost,
		"total_cost":    totalCost,
		"latency_ms":    latencyMs,
		"http_status":   200,
	}

	return s.db.Model(&models.LLMRequestLog{}).Where("id = ?", logID).Updates(updates).Error
}

// extractUsageFromSSE extracts usage information from SSE events
func (s *LLMService) extractUsageFromSSE(data []byte) *models.ChatCompletionUsage {
	// Look for usage_update events injected by our processing
	dataStr := string(data)
	if !strings.Contains(dataStr, "usage_update") {
		return nil
	}

	// Parse the SSE event
	lines := strings.Split(strings.TrimSpace(dataStr), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")

			var usageEvent struct {
				Type  string `json:"type"`
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
					TotalTokens  int `json:"total_tokens"`
				} `json:"usage"`
			}

			if err := json.Unmarshal([]byte(jsonData), &usageEvent); err == nil {
				if usageEvent.Type == "usage_update" {
					return &models.ChatCompletionUsage{
						InputTokens:      usageEvent.Usage.InputTokens,
						OutputTokens:     usageEvent.Usage.OutputTokens,
						TotalTokens:      usageEvent.Usage.TotalTokens,
						PromptTokens:     usageEvent.Usage.InputTokens,  // For OpenAI compatibility
						CompletionTokens: usageEvent.Usage.OutputTokens, // For OpenAI compatibility
					}
				}
			}
		}
	}

	return nil
}

// getDailyUsage 和 getMonthlyUsage 已被 getBatchUsage 替代
// 这些函数已移除，因为新的 getBatchUsage 方法使用单个查询获取所有使用量数据，性能更好

// GetDB returns the database instance for external access
func (s *LLMService) GetDB() *gorm.DB {
	return s.db
}

// ValidateAPIKeyOptimized - 优化版本的API Key验证，使用专门的缓存服务层
func (s *LLMService) ValidateAPIKeyOptimized(apiKey string) (*models.LLMRequestContext, error) {
	ctx := context.Background()

	// 1. 尝试从缓存服务获取API Key信息
	if cachedAPIKey, found := s.cache.GetAPIKey(ctx, apiKey); found {
		// 缓存命中，直接验证使用限制
		return s.validateUsageLimitsOptimized(cachedAPIKey)
	}

	// 2. 缓存未命中，从数据库查询
	var dbAPIKey models.APIKey
	err := s.db.Preload("User").Preload("Provider").First(&dbAPIKey, "key_value = ? AND status = ?", apiKey, "active").Error
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// 3. 检查API Key是否过期（业务逻辑验证）
	if dbAPIKey.ExpiresAt != nil && dbAPIKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	// 4. 将API Key信息存入缓存（通过缓存服务）
	if err := s.cache.SetAPIKey(ctx, apiKey, &dbAPIKey, 5*time.Minute); err != nil {
		// 缓存失败不影响主流程，只记录日志
		fmt.Printf("Failed to cache API key: %v\n", err)
	}

	// 5. 验证使用限制
	return s.validateUsageLimitsOptimized(&dbAPIKey)
}

// validateUsageLimitsOptimized - 优化的使用限制验证，使用缓存服务
func (s *LLMService) validateUsageLimitsOptimized(dbAPIKey *models.APIKey) (*models.LLMRequestContext, error) {
	ctx := context.Background()

	// 批量检查日使用量和月使用量
	if dbAPIKey.DailyRequestLimit > 0 || dbAPIKey.MonthlyRequestLimit > 0 {
		// 1. 尝试从缓存获取使用量统计
		var usage *UsageCounts
		var err error

		if cachedUsage, found := s.cache.GetUsageCount(ctx, dbAPIKey.ID); found {
			usage = cachedUsage
		} else {
			// 2. 缓存未命中，执行批量查询
			usage, err = s.getBatchUsageFromDB(dbAPIKey.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to check usage limits: %w", err)
			}

			// 3. 将使用量存入缓存
			if err := s.cache.SetUsageCount(ctx, dbAPIKey.ID, usage, 1*time.Minute); err != nil {
				fmt.Printf("Failed to cache usage count: %v\n", err)
			}
		}

		// 检查日限制
		if dbAPIKey.DailyRequestLimit > 0 && usage.DailyCount >= int64(dbAPIKey.DailyRequestLimit) {
			return nil, fmt.Errorf("daily request limit exceeded")
		}

		// 检查月限制
		if dbAPIKey.MonthlyRequestLimit > 0 && usage.MonthlyCount >= int64(dbAPIKey.MonthlyRequestLimit) {
			return nil, fmt.Errorf("monthly request limit exceeded")
		}
	}

	// 创建请求上下文
	requestCtx := &models.LLMRequestContext{
		RequestID: uuid.New().String(),
		UserID:    dbAPIKey.UserID,
		APIKeyID:  dbAPIKey.ID,
		Provider:  &dbAPIKey.Provider,
		APIKey:    dbAPIKey,
		StartTime: time.Now(),
	}

	return requestCtx, nil
}

// UsageCounts 使用量统计结构
type UsageCounts struct {
	DailyCount   int64
	MonthlyCount int64
}

// getBatchUsageFromDB - 从数据库批量获取使用量统计，使用单个UNION查询
func (s *LLMService) getBatchUsageFromDB(apiKeyID uint) (*UsageCounts, error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	var results []struct {
		IsDaily bool
		Count   int64
	}

	// 使用UNION查询同时获取日、月使用量
	query := `
		SELECT true as is_daily, COUNT(*) as count
		FROM llm_request_logs 
		WHERE api_key_id = ? AND created_at >= ? AND created_at < ?
		UNION ALL
		SELECT false as is_daily, COUNT(*) as count  
		FROM llm_request_logs
		WHERE api_key_id = ? AND created_at >= ? AND created_at < ?
	`

	err := s.db.Raw(query, apiKeyID, today, tomorrow, apiKeyID, monthStart, monthEnd).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	usage := &UsageCounts{}
	for _, result := range results {
		if result.IsDaily {
			usage.DailyCount = result.Count
		} else {
			usage.MonthlyCount = result.Count
		}
	}

	return usage, nil
}
