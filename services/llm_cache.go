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
// 推荐使用 SetAPIKeyWithUsage 或 SetBatch 进行批量操作以提高性能
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
// 推荐使用 SetAPIKeyWithUsage 或 SetBatch 进行批量操作以提高性能
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

// InvalidateAPIKeyAndUsage 同时使API Key和使用量缓存失效
func (c *CacheService) InvalidateAPIKeyAndUsage(ctx context.Context, apiKey string, apiKeyID uint) error {
	if c.redis == nil {
		return nil
	}

	keys := []string{
		fmt.Sprintf("api_key:%s", apiKey),
		fmt.Sprintf("usage:%d", apiKeyID),
	}

	return c.InvalidateBatch(ctx, keys)
}

// BatchCacheEntry 批量缓存条目
type BatchCacheEntry struct {
	Key   string
	Value interface{}
	TTL   time.Duration
}

// SetBatch 批量设置缓存，使用Redis Pipeline减少网络往返
func (c *CacheService) SetBatch(ctx context.Context, entries []BatchCacheEntry) error {
	if c.redis == nil {
		return nil
	}

	if len(entries) == 0 {
		return nil
	}

	// 使用Pipeline批量操作
	pipe := c.redis.Pipeline()

	for _, entry := range entries {
		data, err := json.Marshal(entry.Value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache entry %s: %w", entry.Key, err)
		}
		pipe.Set(ctx, entry.Key, data, entry.TTL)
	}

	// 执行批量操作
	_, err := pipe.Exec(ctx)
	return err
}

// SetAPIKeyWithUsage 原子性地设置API Key和其使用量统计
func (c *CacheService) SetAPIKeyWithUsage(ctx context.Context, apiKey string, dbAPIKey *models.APIKey, usage *UsageCounts, apiKeyTTL, usageTTL time.Duration) error {
	if c.redis == nil {
		return nil
	}

	// 准备API Key缓存条目
	apiKeyEntry := APIKeyCacheEntry{
		APIKey:    *dbAPIKey,
		CachedAt:  time.Now(),
		ExpiresAt: dbAPIKey.ExpiresAt,
	}

	// 准备使用量缓存条目
	usageEntry := UsageCacheEntry{
		DailyCount:   usage.DailyCount,
		MonthlyCount: usage.MonthlyCount,
		CachedAt:     time.Now(),
	}

	// 批量操作
	entries := []BatchCacheEntry{
		{
			Key:   fmt.Sprintf("api_key:%s", apiKey),
			Value: apiKeyEntry,
			TTL:   apiKeyTTL,
		},
		{
			Key:   fmt.Sprintf("usage:%d", dbAPIKey.ID),
			Value: usageEntry,
			TTL:   usageTTL,
		},
	}

	return c.SetBatch(ctx, entries)
}

// GetBatch 批量获取缓存，减少网络往返
func (c *CacheService) GetBatch(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// 使用Pipeline批量获取
	pipe := c.redis.Pipeline()

	for _, key := range keys {
		pipe.Get(ctx, key)
	}

	results, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]interface{})

	for i, result := range results {
		if cmd, ok := result.(*redis.StringCmd); ok {
			value, err := cmd.Result()
			if err == nil {
				resultMap[keys[i]] = value
			}
		}
	}

	return resultMap, nil
}

// InvalidateBatch 批量删除缓存
func (c *CacheService) InvalidateBatch(ctx context.Context, keys []string) error {
	if c.redis == nil {
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	// 使用Pipeline批量删除
	pipe := c.redis.Pipeline()

	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// WarmupAPIKeyCache 预热API Key缓存 - 优化版本使用批量操作
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

	// 准备批量缓存条目
	var entries []BatchCacheEntry

	for _, apiKey := range apiKeys {
		entry := APIKeyCacheEntry{
			APIKey:    apiKey,
			CachedAt:  time.Now(),
			ExpiresAt: apiKey.ExpiresAt,
		}

		entries = append(entries, BatchCacheEntry{
			Key:   fmt.Sprintf("api_key:%s", apiKey.KeyValue),
			Value: entry,
			TTL:   5 * time.Minute,
		})
	}

	// 批量设置缓存
	if len(entries) > 0 {
		if err := c.SetBatch(ctx, entries); err != nil {
			return fmt.Errorf("failed to batch cache API keys: %w", err)
		}
	}

	return nil
}
