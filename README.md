# Data Splitter

A Go-based tool for automated database archiving with yearly data splitting. This tool migrates data from source tables to year-specific archive databases while maintaining table schemas and relationships.

## Features

- **Year-based Data Splitting**: Automatically splits data by year using configurable date columns
- **Schema Synchronization**: Copies table structures to archive databases
- **Batch Processing**: Configurable batch sizes for efficient data migration
- **Transaction Safety**: Validates migrations and supports rollback on failure
- **Configuration-Driven**: YAML-based configuration for flexible setup
- **Dry Run Support**: Test configurations without modifying data
- **Multi-table Support**: Process multiple tables with different configurations
- **make-sync Integration**: Designed to work with make-sync CI/CD pipelines

## Prerequisites

- Go 1.19+
- MariaDB/MySQL database
- Access to source and archive databases

## Installation

1. Clone the repository
2. Build the application:
   ```bash
   go build -o data-splitter ./cmd
   ```

## Configuration

Create a `config.yaml` file in the project root:

```yaml
# Data Splitter Configuration
version: "1.0"

# Database connection
database:
  type: "mysql"  # mysql, postgres, sqlite
  host: "localhost"
  port: 3306
  user: "your_user"
  password: "your_password"
  source_db: "source_database"

# Tables to sync
tables:
  - name: "user_documents"
    enabled: true
    split_column: "created_at"  # Column for YEAR() filtering
    archive_pattern: "archive_{year}"  # Archive DB naming pattern
  - name: "logs"
    enabled: false  # Skip this table
    split_column: "timestamp"
    archive_pattern: "logs_{year}"

# Archive settings
archive:
  years:
    - 2020
    - 2021
    - 2022
    - 2023
    - 2024
  options:
    batch_size: 1000
    delete_after_archive: true  # Delete from source after successful migration
    create_archive_db: true    # Auto-create archive databases
    dry_run: false             # Test run without actual data changes

# Processing
processing:
  workers: 4  # Concurrent workers
  log_level: "info"  # debug, info, warn, error
  continue_on_error: false  # Stop on first error or continue
```

### Configuration Parameters

#### Database
- `type`: Database type (currently only "mysql" supported)
- `host`: Database host
- `port`: Database port
- `user`: Database username
- `password`: Database password
- `source_db`: Source database name

#### Tables
- `name`: Table name to process
- `enabled`: Whether to process this table
- `split_column`: Date/datetime column for year filtering
- `archive_pattern`: Pattern for archive database names (use `{year}` placeholder)

#### Archive
- `years`: List of years to process
- `options.batch_size`: Number of rows to process in each batch
- `options.delete_after_archive`: Remove data from source after successful migration
- `options.create_archive_db`: Automatically create archive databases
- `options.dry_run`: Test configuration without modifying data

#### Processing
- `workers`: Number of concurrent workers (future enhancement)
- `log_level`: Logging verbosity
- `continue_on_error`: Continue processing other tables/years on error

## Usage

### Basic Usage

```bash
# Run with default config.yaml
./data-splitter

# Run with custom config file
./data-splitter -config custom-config.yaml
```

### Dry Run (Recommended First)

Set `dry_run: true` in config.yaml to test without modifying data:

```bash
./data-splitter
```

This will:
- Validate configuration
- Test database connections
- Log what would be processed
- Exit without making changes

### Production Run

After successful dry run:

1. Set `dry_run: false` in config.yaml
2. Run the migration:
   ```bash
   ./data-splitter
   ```

## make-sync Integration

This tool is designed to work with make-sync pipelines. Example pipeline configuration:

```yaml
# .sync_pipelines/archive-data.yaml
version: "1.0"

jobs:
  prepare:
    commands:
      - "cd data-splitter && go build -o data-splitter ./cmd"
    artifacts:
      - "data-splitter"

  archive:
    dependencies: ["prepare"]
    commands:
      - "cd data-splitter && ./data-splitter"
    environment:
      - "LOG_LEVEL=info"
```

## How It Works

1. **Configuration Loading**: Reads and validates YAML configuration
2. **Database Connection**: Establishes connections to source database
3. **Table Processing**: For each enabled table:
   - Gets table schema from source
   - Creates archive database (if configured)
   - Creates table structure in archive database
   - Migrates data year by year using batch processing
   - Validates migration success
   - Optionally deletes migrated data from source

## Logging

The application uses structured logging with the following levels:
- `debug`: Detailed operation information
- `info`: General progress and status
- `warn`: Non-critical issues
- `error`: Errors that may affect operation
- `fatal`: Critical errors that stop execution

Set log level via `LOG_LEVEL` environment variable or `processing.log_level` config.

## Error Handling

- **Connection Errors**: Fail fast on database connection issues
- **Schema Errors**: Stop processing if table schema cannot be retrieved
- **Migration Errors**: Validate each migration and rollback on failure
- **Validation Errors**: Check row counts between source and archive

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check database host, port, credentials
2. **Table Not Found**: Verify table names in configuration
3. **Column Not Found**: Ensure `split_column` exists and is date/datetime type
4. **Permission Denied**: Check database user privileges for CREATE DATABASE, INSERT, DELETE

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./data-splitter
```

## Architecture

```
cmd/
  main.go                 # Application entry point
internal/
  config/
    config.go            # Configuration loading and validation
  database/
    connection.go        # Database connection management
    schema.go           # Schema discovery and table creation
    migration.go        # Data migration logic
pkg/
  types/
    types.go            # Configuration data structures
```

## Future Enhancements

- Support for PostgreSQL and SQLite
- Concurrent processing of multiple tables
- Compression of archived data
- Automated backup before migration
- Web UI for monitoring and configuration
- REST API for programmatic control

## License

[Your License Here]