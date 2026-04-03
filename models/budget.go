package models

type Budget struct {
	ID          uint     `gorm:"primaryKey"`
	UserID      uint     `gorm:"not null;index"`
	CategoryID  uint     `gorm:"not null"`
	LimitAmount float64  `gorm:"type:decimal(15,2);not null"`
	Month       int      `gorm:"not null"`
	Year        int      `gorm:"not null"`
	User        User     `gorm:"foreignKey:UserID"`
	Category    Category `gorm:"foreignKey:CategoryID"`
}
