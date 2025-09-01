package services

import (
	"fmt"
	"llm-inferra/internal/models"

	"gorm.io/gorm"
)

type ProviderService struct {
	db *gorm.DB
}

func NewProviderService(db *gorm.DB) *ProviderService {
	return &ProviderService{db: db}
}

func (s *ProviderService) List(offset, limit int) ([]models.Provider, int64, error) {
	var providers []models.Provider
	var total int64

	// Count total providers
	if err := s.db.Model(&models.Provider{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated providers with their models
	if err := s.db.Preload("Models").
		Offset(offset).Limit(limit).
		Order("name ASC").
		Find(&providers).Error; err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

func (s *ProviderService) GetByID(id uint) (*models.Provider, error) {
	var provider models.Provider
	err := s.db.Preload("Models").First(&provider, id).Error
	return &provider, err
}

func (s *ProviderService) Create(req models.CreateProviderRequest) (*models.Provider, error) {
	provider := models.Provider{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		BaseURL:     req.BaseURL,
		APIVersion:  req.APIVersion,
	}
	err := s.db.Create(&provider).Error
	return &provider, err
}

func (s *ProviderService) Update(id uint, provider *models.Provider) error {
	//var provider models.Provider
	if err := s.db.Save(provider).Error; err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	return nil
}

func (s *ProviderService) Delete(id uint) error {
	return s.db.Delete(&models.Provider{}, id).Error
}
