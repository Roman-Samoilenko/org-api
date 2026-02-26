package domain

import "time"

type Employee struct {
	ID           uint        `gorm:"primaryKey"          json:"id"`
	DepartmentID uint        `gorm:"not null;index"      json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID" json:"-"`
	FullName     string      `gorm:"size:200;not null"   json:"full_name"`
	Position     string      `gorm:"size:200;not null"   json:"position"`
	HiredAt      *time.Time  `gorm:"index"               json:"hired_at,omitempty"`
	CreatedAt    time.Time   `gorm:"autoCreateTime"      json:"created_at"`
}
