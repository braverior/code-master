package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AnalysisResult struct {
	Modules            []AnalysisModule   `json:"modules"`
	TechStack          []string           `json:"tech_stack"`
	EntryPoints        []string           `json:"entry_points"`
	DirectoryStructure string             `json:"directory_structure"`
	CodeStyle          AnalysisCodeStyle  `json:"code_style"`
}

type AnalysisModule struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	FilesCount  int    `json:"files_count"`
}

type AnalysisCodeStyle struct {
	Naming        string `json:"naming"`
	ErrorHandling string `json:"error_handling"`
	TestFramework string `json:"test_framework"`
}

type JSONAnalysisResult struct {
	Data *AnalysisResult
}

func (j JSONAnalysisResult) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	b, err := json.Marshal(j.Data)
	return string(b), err
}

func (j *JSONAnalysisResult) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	}
	var result AnalysisResult
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	j.Data = &result
	return nil
}

type Repository struct {
	ID                uint               `gorm:"primaryKey" json:"id"`
	ProjectID         uint               `gorm:"not null;index:idx_project_id" json:"project_id"`
	Name              string             `gorm:"type:varchar(128);not null" json:"name"`
	GitURL            string             `gorm:"type:varchar(512);not null" json:"git_url"`
	Platform          string             `gorm:"type:varchar(10);not null" json:"platform"`
	PlatformProjectID string             `gorm:"type:varchar(64)" json:"platform_project_id,omitempty"`
	DefaultBranch     string             `gorm:"type:varchar(64);default:develop" json:"default_branch"`
	AccessToken       string             `gorm:"type:varchar(512)" json:"-"`
	AnalysisResult    JSONAnalysisResult `gorm:"type:json" json:"analysis_result,omitempty"`
	AnalysisStatus    string             `gorm:"type:varchar(20);default:pending" json:"analysis_status"`
	AnalysisError     string             `gorm:"type:text" json:"analysis_error,omitempty"`
	AnalyzedAt        *time.Time         `json:"analyzed_at"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	DeletedAt         gorm.DeletedAt     `gorm:"index" json:"-"`

	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (Repository) TableName() string { return "repositories" }
