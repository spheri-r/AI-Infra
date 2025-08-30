package services

import (
	"llm-inferra/internal/models"

	"gorm.io/gorm"
)

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

func (s *APIKeyService) List(userID uint, offset, limit int) ([]models.APIKey, int64, error) {
	var apiKeys []models.APIKey
	var total int64

	// Count total API keys for the user
	if err := s.db.Model(&models.APIKey{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated API keys with provider information
	if err := s.db.Where("user_id = ?", userID).
		Preload("Provider").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, 0, err
	}

	return apiKeys, total, nil
}

func (s *APIKeyService) Create(userID uint, req models.CreateAPIKeyRequest) (*models.APIKey, error) {
	apiKey := models.APIKey{
		UserID:     userID,
		ProviderID: req.ProviderID,
		Name:       req.Name,
		KeyValue:   req.KeyValue, // In production, this should be encrypted
		Status:     models.APIKeyStatusActive,
	}
	err := s.db.Create(&apiKey).Error
	return &apiKey, err
}

func (s *APIKeyService) GetByID(id uint) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := s.db.Preload("Provider").First(&apiKey, id).Error
	return &apiKey, err
}

func (s *APIKeyService) Update(id uint, apiKey *models.APIKey) error {
	return s.db.Save(apiKey).Error
}

func (s *APIKeyService) Delete(id uint) error {
	return s.db.Delete(&models.APIKey{}, id).Error
}
