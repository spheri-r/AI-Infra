package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"llm-inferra/internal/models"
	"llm-inferra/internal/services"

	"github.com/gin-gonic/gin"
)

type LLMHandler struct {
	llmService *services.LLMService
}

func NewLLMHandler(llmService *services.LLMService) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
	}
}

// extractAPIKey extracts API key from Authorization header or query parameter
func (h *LLMHandler) extractAPIKey(c *gin.Context) string {
	// Try Authorization header first (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Try x-api-key header (common for Anthropic style)
	apiKey := c.GetHeader("x-api-key")
	if apiKey != "" {
		return apiKey
	}

	// Try query parameter as fallback
	return c.Query("api_key")
}

// ChatCompletion handles POST /v1/chat/completions
func (h *LLMHandler) ChatCompletion(c *gin.Context) {
	// Extract API key
	apiKey := h.extractAPIKey(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "API key is required",
				"type":    "authentication_error",
			},
		})
		return
	}

	// Validate API key and get context (using optimized version)
	ctx, err := h.llmService.ValidateAPIKeyOptimized(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "authentication_error",
			},
		})
		return
	}

	// Parse request body
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Invalid request body: %v", err),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Get client information
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Handle streaming vs non-streaming
	if req.Stream {
		h.handleStreamingCompletion(c, ctx, &req, clientIP, userAgent)
	} else {
		h.handleRegularCompletion(c, ctx, &req, clientIP, userAgent)
	}
}

func (h *LLMHandler) handleRegularCompletion(c *gin.Context, ctx *models.LLMRequestContext, req *models.ChatCompletionRequest, clientIP, userAgent string) {
	// Make the completion request
	response, err := h.llmService.ChatCompletion(ctx, req, clientIP, userAgent)
	if err != nil {
		// Determine error type and status code
		statusCode := http.StatusInternalServerError
		errorType := "api_error"

		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "validation") {
			statusCode = http.StatusBadRequest
			errorType = "invalid_request_error"
		} else if strings.Contains(err.Error(), "limit exceeded") {
			statusCode = http.StatusTooManyRequests
			errorType = "rate_limit_error"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorType = "invalid_request_error"
		}

		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    errorType,
			},
		})
		return
	}

	// Return successful response
	c.JSON(http.StatusOK, response)
}

func (h *LLMHandler) handleStreamingCompletion(c *gin.Context, ctx *models.LLMRequestContext, req *models.ChatCompletionRequest, clientIP, userAgent string) {
	// Set headers for Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Get streaming response
	streamChan, err := h.llmService.StreamChatCompletion(ctx, req, clientIP, userAgent)
	if err != nil {
		// Send error as SSE event
		errorType := "api_error"
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "validation") {
			errorType = "invalid_request_error"
		} else if strings.Contains(err.Error(), "limit exceeded") {
			errorType = "rate_limit_error"
		}

		errorEvent := fmt.Sprintf("data: {\"error\": {\"message\": \"%s\", \"type\": \"%s\"}}\n\n", err.Error(), errorType)
		c.String(http.StatusOK, errorEvent)
		return
	}

	// Stream the response
	for {
		select {
		case data, ok := <-streamChan:
			if !ok {
				// Stream closed, send final event
				c.SSEvent("", "[DONE]")
				return
			}

			// Forward the data as-is (Anthropic sends proper SSE format)
			c.Writer.Write(data)
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

// Models endpoint - list available models
func (h *LLMHandler) ListModels(c *gin.Context) {
	// Extract API key
	apiKey := h.extractAPIKey(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "API key is required",
				"type":    "authentication_error",
			},
		})
		return
	}

	// Validate API key (using optimized version)
	ctx, err := h.llmService.ValidateAPIKeyOptimized(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "authentication_error",
			},
		})
		return
	}

	// Get available models for the provider
	var llmModels []models.LLMModel
	if err := h.llmService.GetDB().Where("provider_id = ? AND status = ?", ctx.Provider.ID, models.ModelStatusActive).Find(&llmModels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve models",
				"type":    "api_error",
			},
		})
		return
	}

	// Transform to OpenAI-compatible format
	var responseModels []gin.H
	for _, model := range llmModels {
		responseModels = append(responseModels, gin.H{
			"id":       model.ModelID,
			"object":   "model",
			"created":  model.CreatedAt.Unix(),
			"owned_by": ctx.Provider.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   responseModels,
	})
}

// Health check for LLM service
func (h *LLMHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "llm-proxy",
		"version": "1.0.0",
	})
}
