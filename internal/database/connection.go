package database

import (
	"fmt"
	"log"
	"strings"

	"data-splitter/pkg/types"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectSourceDB establishes connection to the source database
func ConnectSourceDB(config *types.Database) (*gorm.DB, error) {
	dsn := buildDSN(config, config.SourceDB)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
	}

	log.Printf("Connected to source database: %s", config.SourceDB)
	return db, nil
}

// ConnectArchiveDB establishes connection to the archive database
func ConnectArchiveDB(config *types.Database, table *types.Table, year int) (*gorm.DB, error) {
	archiveDB := BuildArchiveDBName(table.ArchivePattern, year)
	dsn := buildDSN(config, archiveDB)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to archive database %s: %w", archiveDB, err)
	}

	log.Printf("Connected to archive database: %s", archiveDB)
	return db, nil
}

// BuildArchiveDBName constructs the archive database name from pattern
func BuildArchiveDBName(pattern string, year int) string {
	// Replace {year} placeholder with actual year
	return strings.Replace(pattern, "{year}", fmt.Sprintf("%d", year), -1)
}

// buildDSN constructs the database connection string
func buildDSN(config *types.Database, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&sql_mode=STRICT_ALL_TABLES",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		database,
	)
}

// TestConnection tests the database connection
func TestConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// CloseConnection closes the database connection
func CloseConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}
