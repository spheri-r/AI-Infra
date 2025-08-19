package services

import (
	"llm-inferra/internal/models"
	"time"

	"gorm.io/gorm"
)

type AnalyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (s *AnalyticsService) GetOverview() (*models.UsageAnalytics, error) {
	// Get overview analytics from the last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var usageLogs []models.UsageLog
	if err := s.db.Preload("User").Preload("APIKey").Preload("Model").Preload("Model.Provider").
		Where("created_at >= ?", thirtyDaysAgo).
		Order("created_at DESC").
		Find(&usageLogs).Error; err != nil {
		return nil, err
	}

	total := int64(len(usageLogs))
	analytics := s.calculateAnalytics(usageLogs, total)

	return analytics, nil
}

func (s *AnalyticsService) GetUsageAnalytics(offset, limit int) (*models.UsageAnalytics, int64, error) {
	var total int64

	// Count total usage logs
	if err := s.db.Model(&models.UsageLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get all usage logs for analytics calculation (not paginated for accurate stats)
	var allUsageLogs []models.UsageLog
	if err := s.db.Preload("User").Preload("APIKey").Preload("Model").
		Order("created_at DESC").
		Find(&allUsageLogs).Error; err != nil {
		return nil, 0, err
	}

	// Calculate comprehensive analytics
	analytics := s.calculateAnalytics(allUsageLogs, total)

	return analytics, total, nil
}

func (s *AnalyticsService) GetCostAnalytics() (*models.UsageAnalytics, error) {
	// Get cost analytics for all time
	var usageLogs []models.UsageLog
	if err := s.db.Preload("User").Preload("APIKey").Preload("Model").Preload("Model.Provider").
		Order("created_at DESC").
		Find(&usageLogs).Error; err != nil {
		return nil, err
	}

	total := int64(len(usageLogs))
	analytics := s.calculateAnalytics(usageLogs, total)

	// Focus on cost-related metrics
	return analytics, nil
}

func (s *AnalyticsService) GetUserAnalytics(offset, limit int) ([]models.UserMetric, int64, error) {
	var userMetrics []models.UserMetric
	var total int64

	// Count total users with usage data
	if err := s.db.Model(&models.User{}).
		Joins("JOIN usage_logs ON users.id = usage_logs.user_id").
		Group("users.id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated user metrics
	if err := s.db.Raw(`
		SELECT 
			u.id as user_id,
			u.username,
			COUNT(ul.id) as requests,
			COALESCE(SUM(ul.total_cost), 0) as cost,
			MAX(ul.created_at) as last_request
		FROM users u 
		LEFT JOIN usage_logs ul ON u.id = ul.user_id 
		GROUP BY u.id, u.username
		ORDER BY cost DESC
		LIMIT ? OFFSET ?
	`, limit, offset).Scan(&userMetrics).Error; err != nil {
		return nil, 0, err
	}

	return userMetrics, total, nil
}

func (s *AnalyticsService) GetProviderAnalytics() ([]models.ProviderMetric, error) {
	var providerMetrics []models.ProviderMetric

	// Get provider analytics using raw SQL for better performance
	if err := s.db.Raw(`
		SELECT 
			p.id as provider_id,
			p.name as provider_name,
			COUNT(ul.id) as requests,
			COALESCE(SUM(ul.total_cost), 0) as cost,
			COALESCE(AVG(CASE WHEN ul.success = true THEN 1.0 ELSE 0.0 END), 0) as success_rate
		FROM providers p 
		LEFT JOIN llm_models lm ON p.id = lm.provider_id
		LEFT JOIN usage_logs ul ON lm.id = ul.model_id 
		GROUP BY p.id, p.name
		HAVING COUNT(ul.id) > 0
		ORDER BY cost DESC
	`).Scan(&providerMetrics).Error; err != nil {
		return nil, err
	}

	return providerMetrics, nil
}

func (s *AnalyticsService) GetModelAnalytics() ([]models.ModelMetric, error) {
	var modelMetrics []models.ModelMetric

	// Get model analytics using raw SQL for better performance
	if err := s.db.Raw(`
		SELECT 
			lm.id as model_id,
			lm.name as model_name,
			COUNT(ul.id) as requests,
			COALESCE(SUM(ul.total_cost), 0) as cost,
			COALESCE(AVG(CASE WHEN ul.success = true THEN 1.0 ELSE 0.0 END), 0) as success_rate
		FROM llm_models lm 
		LEFT JOIN usage_logs ul ON lm.id = ul.model_id 
		GROUP BY lm.id, lm.name
		HAVING COUNT(ul.id) > 0
		ORDER BY cost DESC
	`).Scan(&modelMetrics).Error; err != nil {
		return nil, err
	}

	return modelMetrics, nil
}

func (s *AnalyticsService) GetSystemHealth() (*models.SystemHealth, error) {
	// Calculate system health metrics
	now := time.Now()

	// Get metrics from the last hour
	oneHourAgo := now.Add(-time.Hour)

	var totalRequests, errorCount int64
	var avgResponseTime float64

	// Total requests in the last hour
	if err := s.db.Model(&models.UsageLog{}).
		Where("created_at >= ?", oneHourAgo).
		Count(&totalRequests).Error; err != nil {
		return nil, err
	}

	// Error count in the last hour
	if err := s.db.Model(&models.UsageLog{}).
		Where("created_at >= ? AND success = ?", oneHourAgo, false).
		Count(&errorCount).Error; err != nil {
		return nil, err
	}

	// Average response time in the last hour
	if err := s.db.Model(&models.UsageLog{}).
		Where("created_at >= ?", oneHourAgo).
		Select("AVG(response_time)").
		Scan(&avgResponseTime).Error; err != nil {
		return nil, err
	}

	// Calculate error rate
	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(errorCount) / float64(totalRequests) * 100
	}

	// Calculate requests per second (approximate)
	requestsPerSecond := 0.0
	if totalRequests > 0 {
		requestsPerSecond = float64(totalRequests) / 3600.0 // requests per second in the last hour
	}

	// Count online/offline providers (simplified)
	var onlineProviders, offlineProviders int64
	if err := s.db.Model(&models.Provider{}).
		Where("status = ?", "active").
		Count(&onlineProviders).Error; err != nil {
		onlineProviders = 0
	}

	if err := s.db.Model(&models.Provider{}).
		Where("status != ? OR status IS NULL", "active").
		Count(&offlineProviders).Error; err != nil {
		offlineProviders = 0
	}

	// Get database connection count (simplified)
	var dbConnections int
	if err := s.db.Raw("SELECT COUNT(*) FROM information_schema.processlist").Scan(&dbConnections).Error; err != nil {
		dbConnections = 0 // Fallback for databases that don't support this query
	}

	return &models.SystemHealth{
		Timestamp:           now,
		TotalRequests:       totalRequests,
		RequestsPerSecond:   requestsPerSecond,
		AverageResponseTime: avgResponseTime,
		ErrorRate:           errorRate,
		CPUUsage:            0.0, // Would need system monitoring integration
		MemoryUsage:         0.0, // Would need system monitoring integration
		DiskUsage:           0.0, // Would need system monitoring integration
		DatabaseConnections: dbConnections,
		DatabaseLatency:     0, // Would need to measure actual DB latency
		ProvidersOnline:     int(onlineProviders),
		ProvidersOffline:    int(offlineProviders),
	}, nil
}

func (s *AnalyticsService) GetLogs(offset, limit int) ([]models.UsageLog, int64, error) {
	var logs []models.UsageLog
	var total int64

	// Count total logs
	if err := s.db.Model(&models.UsageLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated logs with all related data
	if err := s.db.Preload("User").Preload("APIKey").Preload("Model").Preload("Model.Provider").
		Offset(offset).Limit(limit).
		Order("created_at DESC"). // Most recent logs first
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// calculateAnalytics computes comprehensive analytics from usage logs
func (s *AnalyticsService) calculateAnalytics(usageLogs []models.UsageLog, total int64) *models.UsageAnalytics {
	if len(usageLogs) == 0 {
		return &models.UsageAnalytics{
			TotalRequests: total,
		}
	}

	// Initialize counters
	var successfulRequests, failedRequests int64
	var totalCost float64
	var totalTokens int64
	var totalResponseTime int64

	// Maps for grouping metrics
	userMetricsMap := make(map[uint]*models.UserMetric)
	providerMetricsMap := make(map[uint]*models.ProviderMetric)
	modelMetricsMap := make(map[uint]*models.ModelMetric)
	dailyMetricsMap := make(map[string]*models.DailyMetric)
	hourlyMetricsMap := make(map[int]*models.HourlyMetric)

	// Process each usage log
	for _, log := range usageLogs {
		// Basic counters
		if log.Success {
			successfulRequests++
		} else {
			failedRequests++
		}
		totalCost += log.TotalCost
		totalTokens += int64(log.TotalTokens)
		totalResponseTime += log.ResponseTime

		// User metrics
		if userMetric, exists := userMetricsMap[log.UserID]; exists {
			userMetric.Requests++
			userMetric.Cost += log.TotalCost
			if log.CreatedAt.After(userMetric.LastRequest) {
				userMetric.LastRequest = log.CreatedAt
			}
		} else {
			userMetricsMap[log.UserID] = &models.UserMetric{
				UserID:      log.UserID,
				Username:    log.User.Username,
				Requests:    1,
				Cost:        log.TotalCost,
				LastRequest: log.CreatedAt,
			}
		}

		// Provider metrics
		if log.Model.Provider.ID != 0 {
			providerID := log.Model.Provider.ID
			if providerMetric, exists := providerMetricsMap[providerID]; exists {
				providerMetric.Requests++
				providerMetric.Cost += log.TotalCost
				if log.Success {
					// Recalculate success rate
					successCount := float64(providerMetric.Requests-1)*providerMetric.SuccessRate + 1
					providerMetric.SuccessRate = successCount / float64(providerMetric.Requests)
				} else {
					// Recalculate success rate
					successCount := float64(providerMetric.Requests-1) * providerMetric.SuccessRate
					providerMetric.SuccessRate = successCount / float64(providerMetric.Requests)
				}
			} else {
				successRate := 0.0
				if log.Success {
					successRate = 1.0
				}
				providerMetricsMap[providerID] = &models.ProviderMetric{
					ProviderID:   providerID,
					ProviderName: log.Model.Provider.Name,
					Requests:     1,
					Cost:         log.TotalCost,
					SuccessRate:  successRate,
				}
			}
		}

		// Model metrics
		if modelMetric, exists := modelMetricsMap[log.ModelID]; exists {
			modelMetric.Requests++
			modelMetric.Cost += log.TotalCost
			if log.Success {
				// Recalculate success rate
				successCount := float64(modelMetric.Requests-1)*modelMetric.SuccessRate + 1
				modelMetric.SuccessRate = successCount / float64(modelMetric.Requests)
			} else {
				// Recalculate success rate
				successCount := float64(modelMetric.Requests-1) * modelMetric.SuccessRate
				modelMetric.SuccessRate = successCount / float64(modelMetric.Requests)
			}
		} else {
			successRate := 0.0
			if log.Success {
				successRate = 1.0
			}
			modelMetricsMap[log.ModelID] = &models.ModelMetric{
				ModelID:     log.ModelID,
				ModelName:   log.Model.Name,
				Requests:    1,
				Cost:        log.TotalCost,
				SuccessRate: successRate,
			}
		}

		// Daily metrics
		dateStr := log.CreatedAt.Format("2006-01-02")
		if dailyMetric, exists := dailyMetricsMap[dateStr]; exists {
			dailyMetric.Requests++
			dailyMetric.Cost += log.TotalCost
			dailyMetric.Tokens += int64(log.TotalTokens)
		} else {
			dailyMetricsMap[dateStr] = &models.DailyMetric{
				Date:     dateStr,
				Requests: 1,
				Cost:     log.TotalCost,
				Tokens:   int64(log.TotalTokens),
			}
		}

		// Hourly metrics
		hour := log.CreatedAt.Hour()
		if hourlyMetric, exists := hourlyMetricsMap[hour]; exists {
			hourlyMetric.Requests++
			hourlyMetric.Cost += log.TotalCost
		} else {
			hourlyMetricsMap[hour] = &models.HourlyMetric{
				Hour:     hour,
				Requests: 1,
				Cost:     log.TotalCost,
			}
		}
	}

	// Calculate average response time
	averageResponseTime := 0.0
	if total > 0 {
		averageResponseTime = float64(totalResponseTime) / float64(total)
	}

	// Convert maps to slices
	userMetrics := s.convertUserMetricsMapToSlice(userMetricsMap)
	providerMetrics := s.convertProviderMetricsMapToSlice(providerMetricsMap)
	modelMetrics := s.convertModelMetricsMapToSlice(modelMetricsMap)
	dailyMetrics := s.convertDailyMetricsMapToSlice(dailyMetricsMap)
	hourlyMetrics := s.convertHourlyMetricsMapToSlice(hourlyMetricsMap)

	return &models.UsageAnalytics{
		TotalRequests:       total,
		SuccessfulRequests:  successfulRequests,
		FailedRequests:      failedRequests,
		TotalCost:           totalCost,
		TotalTokens:         totalTokens,
		AverageResponseTime: averageResponseTime,
		DailyRequests:       dailyMetrics,
		HourlyRequests:      hourlyMetrics,
		ProviderMetrics:     providerMetrics,
		ModelMetrics:        modelMetrics,
		UserMetrics:         userMetrics,
	}
}

// Helper functions to convert maps to slices
func (s *AnalyticsService) convertUserMetricsMapToSlice(userMetricsMap map[uint]*models.UserMetric) []models.UserMetric {
	userMetrics := make([]models.UserMetric, 0, len(userMetricsMap))
	for _, metric := range userMetricsMap {
		userMetrics = append(userMetrics, *metric)
	}
	return userMetrics
}

func (s *AnalyticsService) convertProviderMetricsMapToSlice(providerMetricsMap map[uint]*models.ProviderMetric) []models.ProviderMetric {
	providerMetrics := make([]models.ProviderMetric, 0, len(providerMetricsMap))
	for _, metric := range providerMetricsMap {
		providerMetrics = append(providerMetrics, *metric)
	}
	return providerMetrics
}

func (s *AnalyticsService) convertModelMetricsMapToSlice(modelMetricsMap map[uint]*models.ModelMetric) []models.ModelMetric {
	modelMetrics := make([]models.ModelMetric, 0, len(modelMetricsMap))
	for _, metric := range modelMetricsMap {
		modelMetrics = append(modelMetrics, *metric)
	}
	return modelMetrics
}

func (s *AnalyticsService) convertDailyMetricsMapToSlice(dailyMetricsMap map[string]*models.DailyMetric) []models.DailyMetric {
	dailyMetrics := make([]models.DailyMetric, 0, len(dailyMetricsMap))
	for _, metric := range dailyMetricsMap {
		dailyMetrics = append(dailyMetrics, *metric)
	}
	return dailyMetrics
}

func (s *AnalyticsService) convertHourlyMetricsMapToSlice(hourlyMetricsMap map[int]*models.HourlyMetric) []models.HourlyMetric {
	hourlyMetrics := make([]models.HourlyMetric, 0, len(hourlyMetricsMap))
	for _, metric := range hourlyMetricsMap {
		hourlyMetrics = append(hourlyMetrics, *metric)
	}
	return hourlyMetrics
}
