package handlers

import (
	"net/http"

	"llm-inferra/internal/services"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	overview, err := h.analyticsService.GetOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *AnalyticsHandler) GetUsageAnalytics(c *gin.Context) {
	offset := c.GetInt("offset")
	limit := c.GetInt("limit")

	analytics, total, err := h.analyticsService.GetUsageAnalytics(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics": analytics,
		"total":     total,
		"page":      c.GetInt("page"),
		"limit":     limit,
	})
}

func (h *AnalyticsHandler) GetCostAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetCostAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	offset := c.GetInt("offset")
	limit := c.GetInt("limit")

	analytics, total, err := h.analyticsService.GetUserAnalytics(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_metrics": analytics,
		"total":        total,
		"page":         c.GetInt("page"),
		"limit":        limit,
	})
}

func (h *AnalyticsHandler) GetProviderAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetProviderAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetModelAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetModelAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetSystemHealth(c *gin.Context) {
	health, err := h.analyticsService.GetSystemHealth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, health)
}

func (h *AnalyticsHandler) GetLogs(c *gin.Context) {
	offset := c.GetInt("offset")
	limit := c.GetInt("limit")

	logs, total, err := h.analyticsService.GetLogs(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  c.GetInt("page"),
		"limit": limit,
	})
}
