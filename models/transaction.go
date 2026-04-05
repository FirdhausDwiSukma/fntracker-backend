package models

import "time"

type Transaction struct {
	ID          uint      `gorm:"primaryKey"                    json:"id"`
	UserID      uint      `gorm:"not null;index"                json:"user_id"`
	CategoryID  uint      `gorm:"not null"                      json:"category_id"`
	Amount      float64   `gorm:"type:decimal(15,2);not null"   json:"amount"`
	Type        string    `gorm:"size:10;not null"              json:"type"` // "income" | "expense"
	Description string    `gorm:"type:text"                     json:"description"`
	Date        time.Time `gorm:"type:date;not null"            json:"date"`
	CreatedAt   time.Time `                                     json:"created_at"`
	User        User      `gorm:"foreignKey:UserID"             json:"-"`
	Category    Category  `gorm:"foreignKey:CategoryID"         json:"category,omitempty"`
}
