package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"data-splitter/internal/config"
	"data-splitter/internal/database"
	"data-splitter/pkg/types"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	configPath = flag.String("config", "", "Path to configuration file (default: config.yaml)")
	showInfo   = flag.Bool("info", false, "Show working directory and project directory information")
	projectDir string // Set at build time with -ldflags
)

func main() {
	flag.Parse()

	// Handle --info flag
	if *showInfo {
		displayInfo()
		return
	}

	logrus.Info("Starting Data Splitter")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging with config
	setupLogging(cfg)

	logrus.Infof("Configuration loaded: %d tables, %d years to process", len(cfg.Tables), len(cfg.Archive.Years))

	// Connect to source database
	sourceDB, err := database.ConnectSourceDB(&cfg.Database)
	if err != nil {
		logrus.Fatalf("Failed to connect to source database: %v", err)
	}
	defer database.CloseConnection(sourceDB)

	// Process each enabled table
	totalTables := 0
	for _, table := range cfg.Tables {
		if table.Enabled {
			totalTables++
		}
	}

	logrus.Infof("Starting processing of %d enabled tables", totalTables)

	processedTables := 0
	for _, table := range cfg.Tables {
		if !table.Enabled {
			logrus.Debugf("Skipping disabled table: %s", table.Name)
			continue
		}

		processedTables++
		logrus.Infof("Processing table %d/%d: %s", processedTables, totalTables, table.Name)

		// Process each year for this table
		totalYears := len(cfg.Archive.Years)
		for yearIndex, year := range cfg.Archive.Years {
			logrus.Infof("Processing year %d/%d: %d for table %s", yearIndex+1, totalYears, year, table.Name)

			// Ensure heartbeat interval from processing config is available to archive options
			if cfg.Archive.Options.HeartbeatBatchInterval == 0 {
				cfg.Archive.Options.HeartbeatBatchInterval = cfg.Processing.HeartbeatBatchInterval
			}

			if err := processTableYear(sourceDB, &cfg.Database, &table, year, &cfg.Archive.Options); err != nil {
				// If the error (possibly wrapped) contains a FatalMigrationError,
				// print to stderr and exit non-zero so the pipeline step fails.
				var fmErr database.FatalMigrationError
				if errors.As(err, &fmErr) {
					fmt.Fprintf(os.Stderr, "FATAL: %v\n", fmErr.Error())
					os.Exit(1)
				}
				if cfg.Processing.ContinueOnError {
					logrus.Errorf("Failed to process table %s year %d: %v", table.Name, year, err)
					continue
				} else {
					logrus.Fatalf("Failed to process table %s year %d: %v", table.Name, year, err)
				}
			}

			logrus.Infof("Completed year %d for table %s", year, table.Name)
		}

		logrus.Infof("Completed table %s (%d/%d)", table.Name, processedTables, totalTables)
	}

	logrus.Info("Data Splitter completed successfully")
}

func displayInfo() {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		return
	}

	fmt.Printf("working_dir: %s\n", workingDir)
	fmt.Printf("project_dir: %s\n", projectDir)
}

func setupLogging(config *types.Config) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level from config or default to Info
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = config.Processing.LogLevel
		if level == "" {
			level = "info"
		}
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to 'info'", level)
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)

	// Configure log output
	logPath := config.Processing.LogPath
	if logPath == "" {
		logPath = "logs/data-splitter.log" // Default log file
		logrus.Infof("Using default log path: %s", logPath)
	}

	// Resolve relative paths to working directory
	if !filepath.IsAbs(logPath) {
		// Get current working directory
		wd, err := os.Getwd()
		if err != nil {
			logrus.Warnf("Failed to get working directory: %v, logging to stdout", err)
			return
		}
		logPath = filepath.Join(wd, logPath)
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logrus.Warnf("Failed to create log directory %s: %v, logging to stdout", logDir, err)
		return
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Warnf("Failed to open log file %s: %v, logging to stdout", logPath, err)
		return
	}

	// Set both logrus and standard library log to the same file so that
	// Set both logrus and standard library log to the same file so that
	// interactive spinner/progress output can remain on stdout while all
	// structured logs go to the file. The UI components in `migration.go`
	// (spinner and progress bar) explicitly write to stdout; this prevents
	// interleaving between interactive terminal output and the log file
	// which may be inspected or collected by CI runners.
	logrus.SetOutput(logFile)
	log.SetOutput(logFile)
	logrus.Infof("Logging to file: %s", logPath)
}

func processTableYear(sourceDB *gorm.DB, dbConfig *types.Database, table *types.Table, year int, options *types.ArchiveOptions) error {
	logrus.Infof("Processing table %s for year %d", table.Name, year)

	// Check if dry run
	if options.DryRun {
		logrus.Infof("[DRY RUN] Would process table %s year %d", table.Name, year)
		return nil
	}

	// Connect to archive database
	archiveDB, err := database.ConnectArchiveDB(dbConfig, table, year)
	if err != nil {
		// If archive database doesn't exist and we should create it
		if options.CreateArchiveDB {
			logrus.Infof("Archive database doesn't exist, creating it")
			archiveDBName := database.BuildArchiveDBName(table.ArchivePattern, year)

			if err := database.CreateArchiveDatabase(sourceDB, archiveDBName); err != nil {
				return fmt.Errorf("failed to create archive database: %w", err)
			}

			// Try connecting again
			archiveDB, err = database.ConnectArchiveDB(dbConfig, table, year)
			if err != nil {
				return fmt.Errorf("failed to connect to newly created archive database: %w", err)
			}
		} else {
			return fmt.Errorf("failed to connect to archive database: %w", err)
		}
	}
	defer database.CloseConnection(archiveDB)

	// Get table schema from source
	schema, err := database.GetTableSchema(sourceDB, table.Name)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Create table in archive database
	if err := database.CreateArchiveTable(archiveDB, schema, table.Name); err != nil {
		return fmt.Errorf("failed to create archive table: %w", err)
	}

	// Migrate data
	if err := database.MigrateTableData(sourceDB, archiveDB, table, year, options); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	// Validate migration
	if err := database.ValidateMigration(sourceDB, archiveDB, table, year); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	// Delete migrated data if configured
	if err := database.DeleteMigratedData(sourceDB, table, year, options); err != nil {
		return fmt.Errorf("failed to delete migrated data: %w", err)
	}

	logrus.Infof("Successfully processed table %s for year %d", table.Name, year)
	return nil
}
