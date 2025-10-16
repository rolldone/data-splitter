package types

// Config represents the main configuration structure
type Config struct {
	Version    string     `yaml:"version"`
	Database   Database   `yaml:"database"`
	Tables     []Table    `yaml:"tables"`
	Archive    Archive    `yaml:"archive"`
	Processing Processing `yaml:"processing"`
}

// Database holds database connection configuration
type Database struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SourceDB string `yaml:"source_db"`
}

// Table represents a table to be processed
type Table struct {
	Name           string `yaml:"name"`
	Enabled        bool   `yaml:"enabled"`
	SplitColumn    string `yaml:"split_column"`
	ArchivePattern string `yaml:"archive_pattern"`
}

// Archive holds archive-related settings
type Archive struct {
	Years   []int          `yaml:"years"`
	Options ArchiveOptions `yaml:"options"`
}

// ArchiveOptions holds archive processing options
type ArchiveOptions struct {
	BatchSize          int  `yaml:"batch_size"`
	ResumeOffset       int  `yaml:"resume_offset"`
	DeleteAfterArchive bool `yaml:"delete_after_archive"`
	CreateArchiveDB    bool `yaml:"create_archive_db"`
	DryRun             bool `yaml:"dry_run"`
	// HeartbeatBatchInterval controls how many batches between PROGRESS heartbeats
	// This value is supplied from the top-level processing config when the
	// migration is started.
	HeartbeatBatchInterval int `yaml:"heartbeat_batch_interval"`
}

// Processing holds processing configuration
type Processing struct {
	LogLevel        string `yaml:"log_level"`
	LogPath         string `yaml:"log_path"`
	ContinueOnError bool   `yaml:"continue_on_error"`
	// HeartbeatBatchInterval controls how many batches between PROGRESS heartbeats
	HeartbeatBatchInterval int `yaml:"heartbeat_batch_interval"`
}

// TableInfo holds information about a database table
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// ColumnInfo holds information about a table column
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Key      string
	Default  *string
	Extra    string
}

// MigrationResult holds the result of a migration operation
type MigrationResult struct {
	TableName        string
	Year             int
	RecordsProcessed int
	Success          bool
	Error            error
}
