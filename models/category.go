package models

type Category struct {
	ID     uint   `gorm:"primaryKey"          json:"id"`
	UserID uint   `gorm:"not null;index"      json:"user_id"`
	Name   string `gorm:"size:100;not null"   json:"name"`
	Type   string `gorm:"size:10;not null"    json:"type"` // "income" | "expense"
	User   User   `gorm:"foreignKey:UserID"   json:"-"`
}
