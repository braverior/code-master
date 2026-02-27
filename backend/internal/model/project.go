package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type DocLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type DocLinks []DocLink

func (d DocLinks) Value() (driver.Value, error) {
	if d == nil {
		return "[]", nil
	}
	b, err := json.Marshal(d)
	return string(b), err
}

func (d *DocLinks) Scan(value interface{}) error {
	if value == nil {
		*d = DocLinks{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	}
	return json.Unmarshal(bytes, d)
}

type Project struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(128);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	OwnerID     uint           `gorm:"not null;index:idx_owner_id" json:"owner_id"`
	DocLinks    DocLinks       `gorm:"type:json" json:"doc_links"`
	Status      string         `gorm:"type:varchar(10);default:active;index:idx_status" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Owner   *User            `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []ProjectMember  `gorm:"foreignKey:ProjectID" json:"members,omitempty"`
}

func (Project) TableName() string { return "projects" }
