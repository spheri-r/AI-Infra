package handlers

import (
	"net/http"
	"strconv"

	"llm-inferra/internal/models"
	"llm-inferra/internal/services"

	"github.com/gin-gonic/gin"
)

type ProviderHandler struct {
	providerService *services.ProviderService
}

func NewProviderHandler(providerService *services.ProviderService) *ProviderHandler {
	return &ProviderHandler{providerService: providerService}
}

func (h *ProviderHandler) ListProviders(c *gin.Context) {
	offset := c.GetInt("offset")
	limit := c.GetInt("limit")

	providers, total, err := h.providerService.List(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"total":     total,
		"page":      c.GetInt("page"),
		"limit":     limit,
	})
}

func (h *ProviderHandler) CreateProvider(c *gin.Context) {
	var req models.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.providerService.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

func (h *ProviderHandler) GetProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	provider, err := h.providerService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, provider)
}

func (h *ProviderHandler) UpdateProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.providerService.Update(uint(id), &provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.providerService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

// Placeholder implementations for models - these would need actual implementation
func (h *ProviderHandler) ListModels(c *gin.Context) {
	// TODO: Implement pagination for provider-specific models
	c.JSON(http.StatusOK, gin.H{
		"models": []models.LLMModel{},
		"total":  0,
		"page":   c.GetInt("page"),
		"limit":  c.GetInt("limit"),
	})
}

func (h *ProviderHandler) CreateModel(c *gin.Context) {
	c.JSON(http.StatusCreated, models.LLMModel{})
}

func (h *ProviderHandler) ListAllModels(c *gin.Context) {
	// TODO: Implement pagination for all models
	c.JSON(http.StatusOK, gin.H{
		"models": []models.LLMModel{},
		"total":  0,
		"page":   c.GetInt("page"),
		"limit":  c.GetInt("limit"),
	})
}

func (h *ProviderHandler) GetModel(c *gin.Context) {
	c.JSON(http.StatusOK, models.LLMModel{})
}

func (h *ProviderHandler) UpdateModel(c *gin.Context) {
	c.JSON(http.StatusOK, models.LLMModel{})
}

func (h *ProviderHandler) DeleteModel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Model deleted"})
}
