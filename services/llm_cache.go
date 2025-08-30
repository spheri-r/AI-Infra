package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"llm-inferra/internal/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// CacheService 缓存服务，用于优化数据库查询
type CacheService struct {
	redis *redis.Client
}

// NewCacheService 创建缓存服务
func NewCacheService(redis *redis.Client) *CacheService {
	return &CacheService{redis: redis}
}

// APIKeyCacheEntry API Key缓存条目
type APIKeyCacheEntry struct {
	APIKey    models.APIKey `json:"api_key"`
	CachedAt  time.Time     `json:"cached_at"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
}

// UsageCacheEntry 使用量缓存条目
type UsageCacheEntry struct {
	DailyCount   int64     `json:"daily_count"`
	MonthlyCount int64     `json:"monthly_count"`
	CachedAt     time.Time `json:"cached_at"`
}

// GetAPIKey 从缓存获取API Key
func (c *CacheService) GetAPIKey(ctx context.Context, apiKey string) (*models.APIKey, bool) {
	if c.redis == nil {
		return nil, false
	}

	cacheKey := fmt.Sprintf("api_key:%s", apiKey)
	data, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, false
	}

	var entry APIKeyCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, false
	}

	// 检查是否过期
	if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
		// 删除过期缓存
		c.redis.Del(ctx, cacheKey)
		return nil, false
	}

	return &entry.APIKey, true
}

// SetAPIKey 将API Key存入缓存
func (c *CacheService) SetAPIKey(ctx context.Context, apiKey string, dbAPIKey *models.APIKey, ttl time.Duration) error {
	if c.redis == nil {
		return nil
	}

	entry := APIKeyCacheEntry{
		APIKey:    *dbAPIKey,
		CachedAt:  time.Now(),
		ExpiresAt: dbAPIKey.ExpiresAt,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("api_key:%s", apiKey)
	return c.redis.Set(ctx, cacheKey, data, ttl).Err()
}

// GetUsageCount 从缓存获取使用量统计
func (c *CacheService) GetUsageCount(ctx context.Context, apiKeyID uint) (*UsageCounts, bool) {
	if c.redis == nil {
		return nil, false
	}

	cacheKey := fmt.Sprintf("usage:%d", apiKeyID)
	data, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, false
	}

	var entry UsageCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, false
	}

	usage := &UsageCounts{
		DailyCount:   entry.DailyCount,
		MonthlyCount: entry.MonthlyCount,
	}

	return usage, true
}

// SetUsageCount 将使用量统计存入缓存
func (c *CacheService) SetUsageCount(ctx context.Context, apiKeyID uint, usage *UsageCounts, ttl time.Duration) error {
	if c.redis == nil {
		return nil
	}

	entry := UsageCacheEntry{
		DailyCount:   usage.DailyCount,
		MonthlyCount: usage.MonthlyCount,
		CachedAt:     time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("usage:%d", apiKeyID)
	return c.redis.Set(ctx, cacheKey, data, ttl).Err()
}

// InvalidateAPIKey 使API Key缓存失效
func (c *CacheService) InvalidateAPIKey(ctx context.Context, apiKey string) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("api_key:%s", apiKey)
	return c.redis.Del(ctx, cacheKey).Err()
}

// InvalidateUsageCount 使使用量缓存失效
func (c *CacheService) InvalidateUsageCount(ctx context.Context, apiKeyID uint) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("usage:%d", apiKeyID)
	return c.redis.Del(ctx, cacheKey).Err()
}

// WarmupAPIKeyCache 预热API Key缓存
func (c *CacheService) WarmupAPIKeyCache(ctx context.Context, db *gorm.DB) error {
	if c.redis == nil {
		return nil
	}

	var apiKeys []models.APIKey
	err := db.Preload("User").Preload("Provider").
		Where("status = ?", "active").
		Find(&apiKeys).Error
	if err != nil {
		return err
	}

	for _, apiKey := range apiKeys {
		if err := c.SetAPIKey(ctx, apiKey.KeyValue, &apiKey, 5*time.Minute); err != nil {
			// 记录错误但继续处理其他key
			fmt.Printf("Failed to cache API key %s: %v\n", apiKey.KeyValue, err)
		}
	}

	return nil
}
