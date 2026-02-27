package model

import (
	"time"

	"gorm.io/gorm"
)

type UserSetting struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	BaseURL     string         `gorm:"type:varchar(512)" json:"base_url"`
	APIKey      string         `gorm:"type:varchar(512)" json:"api_key"`
	Model       string         `gorm:"type:varchar(128)" json:"model"`
	GitlabToken string         `gorm:"type:varchar(512)" json:"gitlab_token"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserSetting) TableName() string { return "user_settings" }
