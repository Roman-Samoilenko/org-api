package domain

import "time"

type Employee struct {
	CreatedAt    time.Time   `gorm:"autoCreateTime"          json:"created_at"`
	Department   *Department `gorm:"foreignKey:DepartmentID" json:"-"`
	HiredAt      *time.Time  `gorm:"index"                   json:"hired_at,omitempty"`
	FullName     string      `gorm:"size:200;not null"       json:"full_name"`
	Position     string      `gorm:"size:200;not null"       json:"position"`
	ID           uint        `gorm:"primaryKey"              json:"id"`
	DepartmentID uint        `gorm:"not null;index"          json:"department_id"`
}
