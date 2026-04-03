package models

import "time"

type Transaction struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index"`
	CategoryID  uint      `gorm:"not null"`
	Amount      float64   `gorm:"type:decimal(15,2);not null"`
	Type        string    `gorm:"size:10;not null"` // "income" | "expense"
	Description string    `gorm:"type:text"`
	Date        time.Time `gorm:"type:date;not null"`
	CreatedAt   time.Time
	User        User     `gorm:"foreignKey:UserID"`
	Category    Category `gorm:"foreignKey:CategoryID"`
}
