package config

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global GORM database instance.
var DB *gorm.DB

// ConnectDatabase establishes a PostgreSQL connection via GORM.
// Models passed in the migrate parameter will be auto-migrated.
func ConnectDatabase(dsn string, migrate ...interface{}) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	// Auto-migrate all provided models.
	if len(migrate) > 0 {
		if err := db.AutoMigrate(migrate...); err != nil {
			return nil, err
		}
		log.Println("Database migration completed")
	}

	DB = db
	log.Println("Database connection established")
	return db, nil
}
