package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Get binary location
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "unknown"
	}

	fmt.Println("=== Data Splitter Information ===")
	fmt.Printf("working_dir: %s\n", workingDir)
	fmt.Printf("project_dir: %s\n", projectDir)
	fmt.Printf("current_binary: %s\n", binaryPath)
	fmt.Println()

	// Check if dist folder exists, if not, try to build
	distPath := filepath.Join(projectDir, "dist")
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		fmt.Println("📦 Building cross-platform binaries...")
		if err := buildCrossPlatform(); err != nil {
			fmt.Printf("Warning: Could not build binaries: %v\n", err)
			fmt.Println("Run './build-cross-platform.sh' manually to create dist/ folder")
		}
	}

	// Display available downloads
	fmt.Println("📦 Available Downloads:")
	displayAvailableBinaries(distPath)

	fmt.Println()
	fmt.Println("📦 Installation Locations:")
	fmt.Println("  Linux/macOS: /usr/local/bin/data-splitter")
	fmt.Println("  Windows:     C:\\Program Files\\data-splitter\\data-splitter.exe")
	fmt.Println("               or %USERPROFILE%\\bin\\data-splitter.exe")
	fmt.Println()
	fmt.Println("📁 Configuration & Logs:")
	fmt.Println("  Config file: config.yaml (in working directory)")
	fmt.Println("  Env file:    .env (in working directory)")
	fmt.Println("  Log file:    logs/data-splitter.log (relative to working directory)")
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("  - Place config.yaml and .env in your working directory")
	fmt.Println("  - Run from any directory after installation")
	fmt.Println("  - Use --config flag to specify custom config path")
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

// buildCrossPlatform builds binaries for all supported platforms
func buildCrossPlatform() error {
	platforms := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}

	distPath := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distPath, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	for _, platform := range platforms {
		binaryName := fmt.Sprintf("data-splitter-%s-%s", platform.os, platform.arch)
		if platform.os == "windows" {
			binaryName += ".exe"
		}

		outputPath := filepath.Join(distPath, binaryName)

		cmd := exec.Command("go", "build",
			"-ldflags", fmt.Sprintf("-X main.projectDir=%s", projectDir),
			"-o", outputPath,
			"./cmd")

		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GOOS=%s", platform.os),
			fmt.Sprintf("GOARCH=%s", platform.arch))

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build for %s/%s: %w", platform.os, platform.arch, err)
		}

		fmt.Printf("  ✓ Built %s\n", binaryName)
	}

	return nil
}

// displayAvailableBinaries shows the actual binary files in the dist folder
func displayAvailableBinaries(distPath string) {
	files, err := os.ReadDir(distPath)
	if err != nil {
		fmt.Printf("  No binaries found in %s\n", distPath)
		return
	}

	found := false
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "data-splitter-") {
			fullPath := filepath.Join(distPath, file.Name())
			fmt.Printf("  %s: %s\n", getPlatformName(file.Name()), fullPath)
			found = true
		}
	}

	if !found {
		fmt.Printf("  No data-splitter binaries found in %s\n", distPath)
	}
}

// getPlatformName extracts platform info from binary filename
func getPlatformName(filename string) string {
	// Remove "data-splitter-" prefix and ".exe" suffix
	name := strings.TrimPrefix(filename, "data-splitter-")
	name = strings.TrimSuffix(name, ".exe")

	parts := strings.Split(name, "-")
	if len(parts) >= 2 {
		os := parts[0]
		arch := parts[1]

		switch os {
		case "linux":
			return fmt.Sprintf("Linux (%s)", arch)
		case "darwin":
			if arch == "arm64" {
				return "macOS (Apple Silicon)"
			}
			return fmt.Sprintf("macOS (Intel, %s)", arch)
		case "windows":
			return fmt.Sprintf("Windows (%s)", arch)
		}
	}

	return filename
}
