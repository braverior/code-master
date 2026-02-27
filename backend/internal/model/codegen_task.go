package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type DiffStat struct {
	FilesChanged int        `json:"files_changed"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	Files        []DiffFile `json:"files,omitempty"`
}

type DiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Diff      string `json:"diff,omitempty"`
}

type JSONDiffStat struct {
	Data *DiffStat
}

func (j JSONDiffStat) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	b, err := json.Marshal(j.Data)
	return string(b), err
}

func (j *JSONDiffStat) Scan(value interface{}) error {
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
	var result DiffStat
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	j.Data = &result
	return nil
}

type CodegenTask struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	RequirementID uint         `gorm:"not null;index:idx_requirement_id" json:"requirement_id"`
	RepositoryID  uint         `gorm:"not null" json:"repository_id"`
	SourceBranch  string       `gorm:"type:varchar(64);not null" json:"source_branch"`
	TargetBranch  string       `gorm:"type:varchar(128);not null" json:"target_branch"`
	Status        string       `gorm:"type:varchar(20);default:pending;index:idx_status" json:"status"`
	ExtraContext  string       `gorm:"type:text" json:"extra_context,omitempty"`
	Prompt        string       `gorm:"type:text" json:"prompt,omitempty"`
	OutputLog     string       `gorm:"type:longtext" json:"-"`
	DiffStat      JSONDiffStat `gorm:"type:json" json:"diff_stat,omitempty"`
	CommitSHA     string       `gorm:"type:varchar(64)" json:"commit_sha,omitempty"`
	ErrorMessage  string       `gorm:"type:text" json:"error_message,omitempty"`
	ClaudeCostUSD float64      `gorm:"type:decimal(10,4)" json:"claude_cost_usd,omitempty"`
	PID           int          `gorm:"-" json:"-"`
	StartedAt     *time.Time   `json:"started_at"`
	CompletedAt   *time.Time   `json:"completed_at"`
	CreatedAt     time.Time    `json:"created_at"`

	Requirement *Requirement `gorm:"foreignKey:RequirementID" json:"requirement,omitempty"`
	Repository  *Repository  `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
}

func (CodegenTask) TableName() string { return "codegen_tasks" }
