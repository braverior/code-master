package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type AIReviewResult struct {
	Summary    string              `json:"summary"`
	Issues     []AIReviewIssue     `json:"issues"`
	Categories map[string]AIReviewCategory `json:"categories"`
}

type AIReviewIssue struct {
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	CodeSnippet string `json:"code_snippet"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
}

type AIReviewCategory struct {
	Status  string `json:"status"`
	Details string `json:"details"`
}

type JSONAIReviewResult struct {
	Data *AIReviewResult
}

func (j JSONAIReviewResult) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	b, err := json.Marshal(j.Data)
	return string(b), err
}

func (j *JSONAIReviewResult) Scan(value interface{}) error {
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
	var result AIReviewResult
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	j.Data = &result
	return nil
}

// JSONUintArray stores a JSON array of uint IDs in a single database column.
type JSONUintArray []uint

func (j JSONUintArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONUintArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	}
	return json.Unmarshal(bytes, j)
}

type CodeReview struct {
	ID              uint               `gorm:"primaryKey" json:"id"`
	CodegenTaskID   uint               `gorm:"not null;index:idx_codegen_task_id" json:"codegen_task_id"`
	AIReviewResult  JSONAIReviewResult `gorm:"type:json" json:"ai_review_result,omitempty"`
	AIScore         *int               `json:"ai_score"`
	AIStatus        string             `gorm:"type:varchar(20);default:pending" json:"ai_status"`
	ReviewerIDs     JSONUintArray      `gorm:"type:json" json:"reviewer_ids,omitempty"`
	HumanReviewerID *uint              `gorm:"index:idx_human_reviewer_id" json:"human_reviewer_id"`
	HumanComment    string             `gorm:"type:text" json:"human_comment,omitempty"`
	HumanStatus     string             `gorm:"type:varchar(20);default:pending" json:"human_status"`
	MergeRequestID  string             `gorm:"type:varchar(64)" json:"merge_request_id,omitempty"`
	MergeRequestURL string             `gorm:"type:varchar(512)" json:"merge_request_url,omitempty"`
	MergeStatus     string             `gorm:"type:varchar(10);default:none" json:"merge_status"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`

	CodegenTask   *CodegenTask `gorm:"foreignKey:CodegenTaskID" json:"codegen_task,omitempty"`
	HumanReviewer *User        `gorm:"foreignKey:HumanReviewerID" json:"human_reviewer,omitempty"`
	Reviewers     []*User      `gorm:"-" json:"reviewers,omitempty"`
}

func (CodeReview) TableName() string { return "code_reviews" }
