package model

import (
	"time"

	"gorm.io/gorm"
)

type Requirement struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ProjectID    uint           `gorm:"not null;index:idx_project_id" json:"project_id"`
	Title        string         `gorm:"type:varchar(256);not null" json:"title"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	DocLinks     DocLinks       `gorm:"type:json" json:"doc_links"`
	DocContent   string         `gorm:"type:longtext" json:"-"`
	Priority     string         `gorm:"type:varchar(5);default:p1" json:"priority"`
	Status       string         `gorm:"type:varchar(20);default:draft;index:idx_status" json:"status"`
	Deadline     *time.Time     `json:"deadline"`
	CreatorID    uint           `gorm:"not null;index:idx_creator_id" json:"creator_id"`
	AssigneeID   *uint          `gorm:"index:idx_assignee_id" json:"assignee_id"`
	RepositoryID *uint          `json:"repository_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Creator    *User        `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Assignee   *User        `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Repository *Repository  `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
	Project    *Project     `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (Requirement) TableName() string { return "requirements" }
