package model

import "time"

type ProjectMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;uniqueIndex:uk_project_user" json:"project_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:uk_project_user;index:idx_user_id" json:"user_id"`
	Role      string    `gorm:"type:varchar(10);not null" json:"role"`
	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joined_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ProjectMember) TableName() string { return "project_members" }
