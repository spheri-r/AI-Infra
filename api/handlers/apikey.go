package handlers

import (
	"net/http"
	"strconv"

	"llm-inferra/internal/api/middleware"
	"llm-inferra/internal/models"
	"llm-inferra/internal/services"

	"github.com/gin-gonic/gin"
)

type APIKeyHandler struct {
	apiKeyService *services.APIKeyService
}

func NewAPIKeyHandler(apiKeyService *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	offset := c.GetInt("offset")
	limit := c.GetInt("limit")

	apiKeys, total, err := h.apiKeyService.List(userID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": apiKeys,
		"total":    total,
		"page":     c.GetInt("page"),
		"limit":    limit,
	})
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.apiKeyService.Create(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	apiKey, err := h.apiKeyService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var apiKey models.APIKey
	if err := c.ShouldBindJSON(&apiKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.apiKeyService.Update(uint(id), &apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.apiKeyService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}
