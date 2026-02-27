package service

import (
	"github.com/codeMaster/backend/internal/model"
	"gorm.io/gorm"
)

type SettingService struct {
	db     *gorm.DB
	aesKey string
}

func NewSettingService(db *gorm.DB, aesKey string) *SettingService {
	return &SettingService{db: db, aesKey: aesKey}
}

func (s *SettingService) GetByUserID(userID uint) (*model.UserSetting, error) {
	var setting model.UserSetting
	err := s.db.Where("user_id = ?", userID).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *SettingService) Upsert(userID uint, baseURL, apiKey, modelName, gitlabToken string) (*model.UserSetting, error) {
	var setting model.UserSetting
	err := s.db.Where("user_id = ?", userID).First(&setting).Error

	if err == gorm.ErrRecordNotFound {
		setting = model.UserSetting{
			UserID:      userID,
			BaseURL:     baseURL,
			APIKey:      apiKey,
			Model:       modelName,
			GitlabToken: gitlabToken,
		}
		if err := s.db.Create(&setting).Error; err != nil {
			return nil, err
		}
		return &setting, nil
	}
	if err != nil {
		return nil, err
	}

	setting.BaseURL = baseURL
	setting.APIKey = apiKey
	setting.Model = modelName
	setting.GitlabToken = gitlabToken
	if err := s.db.Save(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}
