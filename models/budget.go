package models

type Budget struct {
	ID          uint     `gorm:"primaryKey"                  json:"id"`
	UserID      uint     `gorm:"not null;index"              json:"user_id"`
	CategoryID  uint     `gorm:"not null"                    json:"category_id"`
	LimitAmount float64  `gorm:"type:decimal(15,2);not null" json:"limit_amount"`
	Month       int      `gorm:"not null"                    json:"month"`
	Year        int      `gorm:"not null"                    json:"year"`
	User        User     `gorm:"foreignKey:UserID"           json:"-"`
	Category    Category `gorm:"foreignKey:CategoryID"       json:"category,omitempty"`
}
