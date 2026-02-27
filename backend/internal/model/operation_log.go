package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
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

type OperationLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index:idx_user_id" json:"user_id"`
	Action       string    `gorm:"type:varchar(64);not null" json:"action"`
	ResourceType string    `gorm:"type:varchar(32);not null;index:idx_resource,priority:1" json:"resource_type"`
	ResourceID   uint      `gorm:"index:idx_resource,priority:2" json:"resource_id"`
	Detail       JSONMap   `gorm:"type:json" json:"detail"`
	IP           string    `gorm:"type:varchar(45)" json:"ip"`
	CreatedAt    time.Time `gorm:"index:idx_created_at" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (OperationLog) TableName() string { return "operation_logs" }
