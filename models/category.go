package models

type Category struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null;index"`
	Name   string `gorm:"size:100;not null"`
	Type   string `gorm:"size:10;not null"` // "income" | "expense"
	User   User   `gorm:"foreignKey:UserID"`
}
