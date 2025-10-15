package database

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// CreateArchiveDatabase creates the archive database if it doesn't exist
func CreateArchiveDatabase(sourceDB *gorm.DB, archiveDBName string) error {
	// Create database if it doesn't exist
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", archiveDBName)
	if err := sourceDB.Exec(createDBSQL).Error; err != nil {
		return fmt.Errorf("failed to create archive database %s: %w", archiveDBName, err)
	}

	log.Printf("Archive database %s created or already exists", archiveDBName)
	return nil
}

// GetTableSchema retrieves the CREATE TABLE statement for a source table
func GetTableSchema(sourceDB *gorm.DB, tableName string) (string, error) {
	var createTableSQL string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)
	row := sourceDB.Raw(query).Row()

	if err := row.Scan(&tableName, &createTableSQL); err != nil {
		return "", fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
	}

	return createTableSQL, nil
}

// CreateArchiveTable creates the table in the archive database or handles existing tables
func CreateArchiveTable(archiveDB *gorm.DB, createTableSQL string, tableName string) error {
	// Check if table already exists
	exists, err := CheckTableExists(archiveDB, tableName)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		log.Printf("Table %s already exists in archive database, skipping creation", tableName)
		return nil
	}

	// Table doesn't exist, create it
	if err := archiveDB.Exec(createTableSQL).Error; err != nil {
		return fmt.Errorf("failed to create table %s in archive database: %w", tableName, err)
	}

	log.Printf("Table %s created in archive database", tableName)
	return nil
}

// GetTableColumns retrieves column information for a table
func GetTableColumns(db *gorm.DB, tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo
	query := fmt.Sprintf("DESCRIBE `%s`", tableName)

	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &col.Default, &col.Extra); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// ColumnInfo represents column information from DESCRIBE
type ColumnInfo struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default interface{}
	Extra   string
}

// BuildSelectQuery builds a SELECT query for data migration with NULLIF for text columns
func BuildSelectQuery(tableName string, splitColumn string, year int, batchSize int, offset int) string {
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE YEAR(`%s`) = %d LIMIT %d OFFSET %d",
		tableName, splitColumn, year, batchSize, offset)
	return query
}

// BuildSelectQueryWithColumns builds a SELECT query with NULLIF transformation for empty strings in text columns
func BuildSelectQueryWithColumns(tableName string, splitColumn string, year int, batchSize int, offset int, columns []ColumnInfo) string {
	var columnSelects []string

	for _, col := range columns {
		colType := strings.ToLower(col.Type)

		// Apply NULLIF for text/longtext/mediumtext columns to convert empty strings to NULL
		// This handles JSON validation constraints that don't allow empty strings
		if strings.Contains(colType, "text") || strings.Contains(colType, "blob") {
			columnSelects = append(columnSelects, fmt.Sprintf("NULLIF(`%s`, '') as `%s`", col.Field, col.Field))
		} else {
			columnSelects = append(columnSelects, fmt.Sprintf("`%s`", col.Field))
		}
	}

	columnList := strings.Join(columnSelects, ", ")
	query := fmt.Sprintf("SELECT %s FROM `%s` WHERE YEAR(`%s`) = %d LIMIT %d OFFSET %d",
		columnList, tableName, splitColumn, year, batchSize, offset)

	return query
}

// BuildInsertQuery builds an INSERT query for data migration
func BuildInsertQuery(tableName string, columns []ColumnInfo) string {
	var columnNames []string
	var placeholders []string

	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", col.Field))
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "))

	return query
}

// GetRowCount gets the total number of rows for a specific year
func GetRowCount(db *gorm.DB, tableName string, splitColumn string, year int) (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE YEAR(`%s`) = %d",
		tableName, splitColumn, year)

	if err := db.Raw(query).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count rows for table %s year %d: %w", tableName, year, err)
	}

	return count, nil
}

// CheckTableExists checks if a table exists in the database
func CheckTableExists(db *gorm.DB, tableName string) (bool, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", tableName)

	if err := db.Raw(query).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}

	return count > 0, nil
}

// CompareTableSchemas compares source and archive table schemas
func CompareTableSchemas(sourceDB *gorm.DB, archiveDB *gorm.DB, tableName string) (bool, error) {
	// Get source table schema
	sourceSchema, err := GetTableSchema(sourceDB, tableName)
	if err != nil {
		return false, fmt.Errorf("failed to get source schema: %w", err)
	}

	// Get archive table schema
	archiveSchema, err := GetTableSchema(archiveDB, tableName)
	if err != nil {
		return false, fmt.Errorf("failed to get archive schema: %w", err)
	}

	// For now, do a simple string comparison
	// In production, you might want more sophisticated schema comparison
	return sourceSchema == archiveSchema, nil
}

// GetPrimaryKeyColumns gets the primary key columns for a table
func GetPrimaryKeyColumns(db *gorm.DB, tableName string) ([]string, error) {
	var primaryKeys []string
	query := fmt.Sprintf("SHOW KEYS FROM `%s` WHERE Key_name = 'PRIMARY'", tableName)

	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys for table %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var table, nonUnique, keyName, seqInIndex, columnName, collation, cardinality, subPart, packed, null, indexType, comment, indexComment string
		var filtered int
		if err := rows.Scan(&table, &nonUnique, &keyName, &seqInIndex, &columnName, &collation, &cardinality, &subPart, &packed, &null, &indexType, &comment, &indexComment, &filtered); err != nil {
			return nil, fmt.Errorf("failed to scan primary key info: %w", err)
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	return primaryKeys, nil
}

// BuildMergeInsertQuery builds an INSERT ... ON DUPLICATE KEY UPDATE query for data migration
func BuildMergeInsertQuery(tableName string, columns []ColumnInfo) (string, error) {
	var columnNames []string
	var placeholders []string
	var updateParts []string

	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", col.Field))
		placeholders = append(placeholders, "?")
		// For merge, update all non-primary key columns
		// Note: This assumes we want to overwrite existing data
		if col.Key != "PRI" {
			updateParts = append(updateParts, fmt.Sprintf("`%s` = VALUES(`%s`)", col.Field, col.Field))
		}
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "))

	if len(updateParts) > 0 {
		query += " ON DUPLICATE KEY UPDATE " + strings.Join(updateParts, ", ")
	}

	return query, nil
}
