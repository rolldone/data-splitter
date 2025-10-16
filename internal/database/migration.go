package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"data-splitter/pkg/types"

	"github.com/briandowns/spinner"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// MigrateTableData migrates data from source table to archive table for a specific year
func MigrateTableData(sourceDB *gorm.DB, archiveDB *gorm.DB, table *types.Table, year int, config *types.ArchiveOptions) error {
	startTime := time.Now()
	log.Printf("Starting data migration for table %s, year %d", table.Name, year)

	// Emit an initial one-line PROGRESS message so pipelines can detect start
	fmt.Printf("PROGRESS table=%s year=%d processed=%d total=%d batch=%d status=started\n", table.Name, year, 0, 0, 0)

	// Get table columns
	columns, err := GetTableColumns(sourceDB, table.Name)
	if err != nil {
		return fmt.Errorf("failed to get columns for table %s: %w", table.Name, err)
	}

	// Get total row count
	totalRows, err := GetRowCount(sourceDB, table.Name, table.SplitColumn, year)
	if err != nil {
		return fmt.Errorf("failed to get row count for table %s: %w", table.Name, err)
	}

	if totalRows == 0 {
		log.Printf("No data found for table %s, year %d", table.Name, year)
		return nil
	}

	log.Printf("Migrating %d rows for table %s, year %d", totalRows, table.Name, year)

	// Setup interactive progress UI (progress bar + optional spinner)
	var sp *spinner.Spinner

	// Spinner enabled by default (both pipeline and local). Disable with NO_SPINNER=1.
	noSpinner := os.Getenv("NO_SPINNER") != ""
	noProgress := os.Getenv("NO_PROGRESS") != ""

	if noProgress {
		log.Printf("Progress disabled via NO_PROGRESS")
	}
	if noSpinner {
		log.Printf("Spinner disabled via NO_SPINNER")
	}

	if !noSpinner {
		// spinner is cosmetic; keep small interval and write to Stdout to avoid
		// interleaving with log file output. Enabled by default for both pipeline
		// and local runs unless NO_SPINNER is set.
		sp = spinner.New(spinner.CharSets[14], 120*time.Millisecond)
		// Write spinner to stderr so pipeline stdout remains available for
		// stable one-line progress messages. Many CI systems capture stdout for
		// step output parsing; writing dynamic spinner chars to stderr avoids
		// corrupting that stream.
		sp.Writer = os.Stderr
		sp.Suffix = fmt.Sprintf(" Loading %s", table.Name)
		sp.Start()
		defer sp.Stop()
	}

	// Build merge insert query (handles existing data)
	insertQuery, err := BuildMergeInsertQuery(table.Name, columns)
	if err != nil {
		return fmt.Errorf("failed to build merge insert query: %w", err)
	}

	// Process data in batches
	offset := config.ResumeOffset
	if offset > 0 {
		log.Printf("Resuming migration from offset %d for table %s, year %d", offset, table.Name, year)
	}
	migratedRows := int64(offset)           // Start counting from resume offset
	batchCount := offset / config.BatchSize // Calculate starting batch number

	log.Printf("Starting batch processing for table %s, year %d: %d total rows, batch size %d, starting from offset %d", table.Name, year, totalRows, config.BatchSize, offset)

	for offset < int(totalRows) {
		batchCount++
		batchSize := config.BatchSize
		if offset+batchSize > int(totalRows) {
			batchSize = int(totalRows) - offset
		}

		log.Printf("Processing batch %d: offset %d, size %d for table %s, year %d", batchCount, offset, batchSize, table.Name, year)

		// Migrate batch
		rowsAffected, err := migrateBatch(sourceDB, archiveDB, table, year, columns, insertQuery, batchSize, offset)
		if err != nil {
			log.Printf("ERROR: Failed to migrate batch %d at offset %d: %v", batchCount, offset, err)
			// Print recent logs to stderr for pipeline visibility
			PrintRecentLogTail(200)
			return FatalMigrationError{Err: fmt.Errorf("failed to migrate batch at offset %d: %w", offset, err)}
		}

		migratedRows += rowsAffected
		offset += batchSize

		// Update UI (spinner suffix only; progress bar removed)
		if sp != nil {
			sp.Suffix = fmt.Sprintf(" Loading %s - %d/%d (batch %d)", table.Name, migratedRows, totalRows, batchCount)
		}

		log.Printf("Completed batch %d: migrated %d/%d rows for table %s, year %d", batchCount, migratedRows, totalRows, table.Name, year)

		// Determine heartbeat interval (configured via archive options; default 10)
		heartbeatInterval := config.HeartbeatBatchInterval
		if heartbeatInterval <= 0 {
			heartbeatInterval = 10
		}

		// Heartbeat every N batches
		if batchCount%heartbeatInterval == 0 {
			// Log heartbeat to log file
			log.Printf("HEARTBEAT: Processed %d batches, %d/%d rows for table %s, year %d", batchCount, migratedRows, totalRows, table.Name, year)
			// Also emit a stable one-line progress message to stdout so pipelines
			// that capture stdout can read progress without dealing with ANSI
			// or carriage returns.
			fmt.Printf("PROGRESS table=%s year=%d processed=%d total=%d batch=%d\n",
				table.Name, year, migratedRows, totalRows, batchCount)
		}
	}

	// progress bar removed; spinner will be stopped by defer

	duration := time.Since(startTime)
	log.Printf("Completed data migration for table %s, year %d: %d rows migrated (duration=%s)", table.Name, year, migratedRows, duration)

	// Emit a final progress line and a FINAL summary so pipelines can detect completion
	fmt.Printf("PROGRESS table=%s year=%d processed=%d total=%d batch=%d status=completed duration=%s\n",
		table.Name, year, migratedRows, totalRows, batchCount, duration)
	// Also emit a concise FINAL line (machine-friendly)
	fmt.Printf("FINAL table=%s year=%d processed=%d duration=%s exit=0\n", table.Name, year, migratedRows, duration)

	return nil
}

// FatalMigrationError marks an error as fatal such that the caller should exit
// immediately (non-zero). We use this for critical failures where continuing
// processing would be unsafe or misleading.
type FatalMigrationError struct {
	Err error
}

func (e FatalMigrationError) Error() string { return e.Err.Error() }
func (e FatalMigrationError) Unwrap() error { return e.Err }

// migrateBatch migrates a single batch of data
func migrateBatch(sourceDB *gorm.DB, archiveDB *gorm.DB, table *types.Table, year int, columns []ColumnInfo, insertQuery string, batchSize int, offset int) (int64, error) {
	log.Printf("DEBUG: Starting migrateBatch - table: %s, year: %d, batchSize: %d, offset: %d", table.Name, year, batchSize, offset)

	// Build select query with NULLIF transformation for text columns
	selectQuery := BuildSelectQueryWithColumns(table.Name, table.SplitColumn, year, batchSize, offset, columns)
	log.Printf("DEBUG: Select query: %s", selectQuery)

	// Execute select query
	log.Printf("DEBUG: Executing select query...")
	rows, err := sourceDB.Raw(selectQuery).Rows()
	if err != nil {
		log.Printf("ERROR: Failed to execute select query: %v", err)
		// print recent logs for pipeline visibility
		PrintRecentLogTail(200)
		return 0, fmt.Errorf("failed to execute select query: %w", err)
	}
	defer rows.Close()

	log.Printf("DEBUG: Select query completed, processing rows...")

	// Prepare values for batch insert
	var batchValues [][]interface{}
	var rowCount int64

	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row into values
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("ERROR: Failed to scan row: %v", err)
			// print recent logs for pipeline visibility
			PrintRecentLogTail(200)
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		batchValues = append(batchValues, values)
		rowCount++
	}

	log.Printf("DEBUG: Processed %d rows from select query", rowCount)

	if rowCount == 0 {
		log.Printf("DEBUG: No rows to process, returning")
		return 0, nil
	}

	// Execute batch insert
	log.Printf("Executing batch insert with %d rows...", rowCount)
	if err := executeBatchInsert(archiveDB, insertQuery, batchValues); err != nil {
		log.Printf("WARNING: Batch insert had errors: %v (some rows may have succeeded)", err)
		// Don't return error here - partial success is acceptable
		// Only return error if ALL rows failed (handled in executeBatchInsert)
		if err.Error() == fmt.Sprintf("all %d rows failed to insert", rowCount) {
			// print recent logs for pipeline visibility
			PrintRecentLogTail(200)
			return 0, fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	log.Printf("Batch insert completed")
	return rowCount, nil
}

// executeBatchInsert executes a batch insert/merge operation with constraint bypass for backup
func executeBatchInsert(db *gorm.DB, query string, batchValues [][]interface{}) error {
	log.Printf("Starting raw SQL batch insert with %d rows (constraint bypass enabled)", len(batchValues))

	// Get raw SQL database connection to bypass GORM constraints
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw database connection: %w", err)
	}

	// Detect database type and apply appropriate constraint bypass
	dbType, err := detectDatabaseType(sqlDB)
	if err != nil {
		log.Printf("WARNING: Failed to detect database type: %v, proceeding with MySQL approach", err)
		dbType = "mysql"
	}

	log.Printf("Detected database type: %s", dbType)

	// Apply database-specific constraint bypass
	switch strings.ToLower(dbType) {
	case "postgresql", "postgres":
		// PostgreSQL: Use session_replication_role to bypass constraints & triggers
		if _, err := sqlDB.Exec("SET session_replication_role = replica"); err != nil {
			log.Printf("WARNING: Failed to disable PostgreSQL constraints: %v", err)
		}
		defer func() {
			if _, err := sqlDB.Exec("SET session_replication_role = origin"); err != nil {
				log.Printf("WARNING: Failed to re-enable PostgreSQL constraints: %v", err)
			}
		}()
	case "sqlite", "sqlite3":
		// SQLite: Disable foreign key constraints (check constraints cannot be disabled)
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = OFF"); err != nil {
			log.Printf("WARNING: Failed to disable SQLite foreign keys: %v", err)
		}
		defer func() {
			if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
				log.Printf("WARNING: Failed to re-enable SQLite foreign keys: %v", err)
			}
		}()
		log.Printf("NOTE: SQLite check constraints cannot be bypassed - invalid data will cause failures")
	case "sqlserver", "mssql":
		// SQL Server: No direct way to disable check constraints globally
		// Alternative: Use INSERT with error handling
		log.Printf("NOTE: SQL Server check constraints cannot be bypassed globally - using error-tolerant mode")
	default:
		// MySQL/MariaDB and others: Use CHECK_CONSTRAINT_CHECKS
		if _, err := sqlDB.Exec("SET CHECK_CONSTRAINT_CHECKS = 0"); err != nil {
			log.Printf("WARNING: Failed to disable MySQL constraints: %v", err)
		}
		defer func() {
			if _, err := sqlDB.Exec("SET CHECK_CONSTRAINT_CHECKS = 1"); err != nil {
				log.Printf("WARNING: Failed to re-enable MySQL constraints: %v", err)
			}
		}()
	}

	// Process each row with raw SQL (bypass all GORM validations)
	inserted := 0
	updated := 0
	failed := 0

	for i, values := range batchValues {
		if i%100 == 0 && i > 0 { // Log progress every 100 rows
			log.Printf("Progress: %d/%d rows (inserted: %d, updated: %d, failed: %d)",
				i, len(batchValues), inserted, updated, failed)
		}

		// Execute raw INSERT with ON DUPLICATE KEY UPDATE (MySQL) or ON CONFLICT (PostgreSQL)
		result, err := sqlDB.Exec(query, values...)
		if err != nil {
			// Log failed inserts but continue (backup mode)
			firstValue := "unknown"
			if len(values) > 0 {
				firstValue = fmt.Sprintf("%v", values[0])
			}

			log.Printf("ERROR: Raw insert failed for row %d (ID: %s): %v", i+1, firstValue, err)
			failed++
			continue
		}

		// Check result
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 1 {
			inserted++
		} else if rowsAffected == 2 {
			updated++
		} else if rowsAffected == 0 {
			// This shouldn't happen with constraints disabled, but log anyway
			log.Printf("WARNING: Row %d had 0 rows affected (ID: %v) - unexpected with constraints disabled", i+1, values[0])
		}
	}

	// Log final summary
	log.Printf("Raw SQL batch insert completed: %d inserted, %d updated, %d failed (total: %d) - constraints bypassed for %s",
		inserted, updated, failed, len(batchValues), dbType)

	return nil
}

// detectDatabaseType detects the database type from connection
func detectDatabaseType(sqlDB *sql.DB) (string, error) {
	var version string
	err := sqlDB.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		// Fallback: try SQLite syntax
		err2 := sqlDB.QueryRow("SELECT sqlite_version()").Scan(&version)
		if err2 == nil {
			return "sqlite", nil
		}
		return "", fmt.Errorf("failed to detect database type: %w", err)
	}

	versionLower := strings.ToLower(version)
	if strings.Contains(versionLower, "postgresql") || strings.Contains(versionLower, "postgres") {
		return "postgresql", nil
	} else if strings.Contains(versionLower, "mariadb") {
		return "mariadb", nil
	} else if strings.Contains(versionLower, "mysql") {
		return "mysql", nil
	} else if strings.Contains(versionLower, "microsoft") || strings.Contains(versionLower, "sql server") {
		return "sqlserver", nil
	} else if strings.Contains(versionLower, "sqlite") {
		return "sqlite", nil
	}

	return "unknown", nil
}

// DeleteMigratedData deletes the migrated data from source table if configured
func DeleteMigratedData(sourceDB *gorm.DB, table *types.Table, year int, config *types.ArchiveOptions) error {
	if !config.DeleteAfterArchive {
		log.Printf("Skipping data deletion for table %s, year %d (delete_after_archive is false)", table.Name, year)
		return nil
	}

	log.Printf("Deleting migrated data for table %s, year %d", table.Name, year)

	deleteQuery := fmt.Sprintf("DELETE FROM `%s` WHERE YEAR(`%s`) = %d", table.Name, table.SplitColumn, year)

	// Run delete with GORM SQL logging silenced to avoid raw SQL being emitted to pipeline logs
	// (some remote runners may add quoting around logged SQL which can cause command failures).
	silentDB := sourceDB.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	result := silentDB.Exec(deleteQuery)
	if result.Error != nil {
		return fmt.Errorf("failed to delete migrated data: %w", result.Error)
	}

	deletedRows := result.RowsAffected
	log.Printf("Deleted %d rows from source table %s, year %d", deletedRows, table.Name, year)

	return nil
}

// ValidateMigration validates that the migration was successful
func ValidateMigration(sourceDB *gorm.DB, archiveDB *gorm.DB, table *types.Table, year int) error {
	// Count rows in source
	sourceCount, err := GetRowCount(sourceDB, table.Name, table.SplitColumn, year)
	if err != nil {
		return fmt.Errorf("failed to count source rows: %w", err)
	}

	// Count rows in archive for this year
	archiveCount, err := GetRowCount(archiveDB, table.Name, table.SplitColumn, year)
	if err != nil {
		return fmt.Errorf("failed to count archive rows: %w", err)
	}

	// For merge operations, we expect at least as many rows in archive as in source
	// (archive might have more from previous runs or other years)
	if archiveCount < sourceCount {
		return fmt.Errorf("migration validation failed: source has %d rows, archive has %d rows (expected at least %d)", sourceCount, archiveCount, sourceCount)
	}

	log.Printf("Migration validation successful for table %s, year %d: source=%d, archive=%d rows", table.Name, year, sourceCount, archiveCount)
	return nil
}
