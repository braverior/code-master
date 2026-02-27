package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	FeishuUID      string         `gorm:"type:varchar(128);uniqueIndex:idx_feishu_uid;not null" json:"-"`
	FeishuUnionID  string         `gorm:"type:varchar(128);uniqueIndex" json:"-"`
	Name           string         `gorm:"type:varchar(64);not null" json:"name"`
	Avatar         string         `gorm:"type:varchar(512)" json:"avatar"`
	Email          string         `gorm:"type:varchar(128)" json:"email"`
	Role           string         `gorm:"type:varchar(10);not null;default:rd;index:idx_role" json:"role"`
	IsAdmin        bool           `gorm:"default:false" json:"is_admin"`
	Status         int            `gorm:"default:1" json:"status"`
	LastLoginAt    *time.Time     `json:"last_login_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

type UserBrief struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar,omitempty"`
	Role    string `json:"role,omitempty"`
	IsAdmin bool   `json:"is_admin"`
	Email   string `json:"email,omitempty"`
}

func (u *User) Brief() UserBrief {
	return UserBrief{
		ID:      u.ID,
		Name:    u.Name,
		Avatar:  u.Avatar,
		Role:    u.Role,
		IsAdmin: u.IsAdmin,
		Email:   u.Email,
	}
}
