package domain

import "time"

type Department struct {
	CreatedAt time.Time    `gorm:"autoCreateTime"      json:"created_at"`
	ParentID  *uint        `gorm:"index"               json:"parent_id,omitempty"`
	Parent    *Department  `gorm:"foreignKey:ParentID" json:"-"`
	Name      string       `gorm:"size:200;not null"   json:"name"`
	Children  []Department `gorm:"-"                   json:"children,omitempty"`
	Employees []Employee   `gorm:"-"                   json:"employees,omitempty"`
	ID        uint         `gorm:"primaryKey"          json:"id"`
}
