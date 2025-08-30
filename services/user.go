package services

import (
	"fmt"

	"llm-inferra/internal/models"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("APIKeys").Preload("UsageLogs").First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (s *UserService) List(offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if err := s.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (s *UserService) Update(id uint, req models.UpdateUserRequest) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Update fields if provided
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.DailyRequestLimit != nil {
		user.DailyRequestLimit = *req.DailyRequestLimit
	}
	if req.MonthlyRequestLimit != nil {
		user.MonthlyRequestLimit = *req.MonthlyRequestLimit
	}
	if req.DailyCostLimit != nil {
		user.DailyCostLimit = *req.DailyCostLimit
	}
	if req.MonthlyCostLimit != nil {
		user.MonthlyCostLimit = *req.MonthlyCostLimit
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

func (s *UserService) Delete(id uint) error {
	if err := s.db.Delete(&models.User{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
