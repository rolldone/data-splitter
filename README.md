# Data Splitter

Panduan singkat untuk membangun, menjalankan, dan memverifikasi UI progress (progress bar + spinner) serta pemisahan log di project `data-splitter`.

Semua instruksi ada dalam Bahasa Indonesia dan fokus pada verifikasi bahwa:
- UI interaktif (progress bar + spinner) menulis ke stdout,
- Semua log terstruktur (logrus + standard log) ditulis ke file log (default `logs/data-splitter.log`) sehingga tidak mengacaukan output pipeline/runner.

Catatan singkat soal perilaku:
- UI akan otomatis dinonaktifkan jika tidak berjalan di TTY atau jika environment `CI=true`.
- Override manual tersedia dengan `NO_SPINNER=1` dan/atau `NO_PROGRESS=1`.

Prasyarat
- Go toolchain terpasang (go 1.18+ direkomendasikan).

1) Build binary

Jalankan dari folder project:

```bash
cd /path/to/data-splitter
go build -o data-splitter ./...
```

2) Run interaktif (lihat progress & spinner)

Jalankan binary seperti biasa (tambahkan `--config` jika repo kamu memakai flag):

```bash
./data-splitter
# atau jika perlu config
./data-splitter --config config.yaml
```

Yang harus diverifikasi:
- Di terminal seharusnya terlihat progress bar yang bergerak dan spinner yang menunjukkan aktivtas.
- File log default ada di `logs/data-splitter.log`. Periksa dengan:

```bash
tail -n 200 logs/data-splitter.log
```

Pastikan log hanya berisi entri structured (Info/Debug/Warning) bukan karakter progress bar.

3) Run non-interactive / CI mode

Untuk mensimulasikan environment CI (UI harus auto-disable):

```bash
CI=true ./data-splitter
```

Atau paksa disable UI dengan env:

```bash
NO_SPINNER=1 NO_PROGRESS=1 ./data-splitter
```

Verifikasi:
- Di terminal seharusnya tidak ada progress bar atau spinner.
- Semua log tetap ada di file `logs/data-splitter.log`.

4) Menangkap stdout untuk melihat apa yang pipeline akan lihat

Jika pipeline capture stdout, jalankan:

```bash
./data-splitter > stdout.capture 2>&1
# interactive run akan menulis UI ke stdout (expected)

CI=true ./data-splitter > stdout.capture 2>&1
# CI run seharusnya punya sedikit atau tidak ada UI artifacts di stdout.capture
```

5) Troubleshooting singkat

- Jika kamu melihat SQL atau keluaran besar lain di stdout, pastikan `setupLogging` di `cmd/main.go` dieksekusi sebelum koneksi DB dibuat.
- Pastikan tidak ada kode yang secara eksplisit menulis log ke stdout (fmt.Print*). Saya sudah memeriksa repo dan tidak menemukan `fmt.Print*` di file utama.

6) Opsional: verifikasi otomatis (skrip)

Saya bisa menambahkan skrip helper `scripts/verify-ui.sh` yang menjalankan langkah-langkah di atas otomatis dan menuliskan ringkasan. Beri tahu jika kamu mau saya tambahkan.

--
Dokumentasi ini menambahkan instruksi verifikasi UI vs log dan env override. Jika kamu ingin saya commit skrip verifikasi otomatis atau menambahkan contoh konfigurasi `config.sample.yaml` yang menampilkan opsi `NO_SPINNER`/`NO_PROGRESS`, beri tahu dan saya akan tambahkan.
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

### Additional runtime signals for CI / parsers

- `LOG_TAIL_LINES` (env): When a fatal error occurs the program prints a tail of the
  structured log file to stderr to aid debugging. `LOG_TAIL_LINES` lets you control how
  many lines are printed. Example: `LOG_TAIL_LINES=50` (default depends on caller; the
  code uses 200 when not overridden by env).

- `PROGRESS` / `FINAL` lines (stdout): The program emits machine-friendly one-line
  progress heartbeats to stdout so CI runners or parsers can track progress without
  being confused by interactive ANSI output. Example lines:

```
PROGRESS table=users year=2025 processed=0 total=0 batch=0 status=started
PROGRESS table=users year=2025 processed=1000 total=96716 batch=10
... (periodic heartbeat)
PROGRESS table=users year=2025 processed=96716 total=96716 batch=97 status=completed duration=1h23m45s
FINAL table=users year=2025 processed=96716 duration=1h23m45s exit=0
```

- `HEARTBEAT_BATCH_INTERVAL` (concept): The heartbeat is emitted every N batches. By
  default the code emits a heartbeat every 10 batches. This value may later be
  configurable via an env var (e.g., `HEARTBEAT_BATCH_INTERVAL`) if you want a
  different frequency for long/short batch sizes. The heartbeat frequency controls
  how often you see `PROGRESS` lines — increasing it reduces stdout churn but
  provides coarser progress updates.

### Using secrets and placeholders in config.yaml

If you want to commit `config.yaml` but keep secrets out of the repo, you can use
OS environment variable placeholders in the YAML and store secrets in a local
`.env` file (not committed). Example in `config.yaml`:

```yaml
database:
  host: "${DB_HOST}"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
```

The loader will automatically load a `.env` file (if present) and expand
`${VAR}` placeholders using the process environment. `.env` values do not
override existing environment variables; this allows CI-provided env to take
precedence.

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