# Data Splitter

Data Splitter adalah alat baris perintah untuk memindahkan data historis dari
tabel sumber ke database arsip per-tahun. Tool ini berguna untuk mengarsipkan
data besar berdasarkan kolom tanggal (mis. created_at), menyalin struktur
tabel ke database arsip, dan melakukan migrasi data dalam batch yang dapat
di-resume. Outputnya dirancang agar aman dipakai di pipeline CI: ringkasan
progres (PROGRESS/FINAL) dicetak ke stdout untuk parsing oleh runner, sementara
log terstruktur lengkap ditulis ke file.

## Fitur Utama

- **Pembagian Data Berdasarkan Tahun**: Secara otomatis membagi data berdasarkan kolom tanggal yang dapat dikonfigurasi
- **Sinkronisasi Skema**: Menyalin struktur tabel ke database arsip
- **Batch Processing**: Ukuran batch yang dapat dikonfigurasi untuk migrasi data yang efisien
- **Keamanan Transaksi**: Memvalidasi migrasi dan mendukung rollback jika gagal
- **Konfigurasi-Driven**: Setup yang fleksibel menggunakan file YAML
- **Dry Run Support**: Test konfigurasi tanpa memodifikasi data
- **Multi-table Support**: Memproses multiple tabel dengan konfigurasi berbeda

## Prasyarat

- Go toolchain terpasang (Go 1.18+ direkomendasikan)

## Build

### Linux/macOS

```bash
cd data-splitter
go build -ldflags "-X main.projectDir=$(pwd)" -o data-splitter ./cmd
```

### Windows

Untuk Windows, gunakan cross-compilation dari Linux/macOS:

```bash
cd data-splitter
./build-cross-platform.sh
```

Atau build langsung di Windows dengan Go:

```cmd
cd data-splitter
go build -ldflags "-X main.projectDir=%cd%" -o data-splitter.exe .\cmd
```

### Cross-platform Build

Script `build-cross-platform.sh` akan membuat binary untuk semua platform:

```bash
./build-cross-platform.sh
```

Binary akan tersedia di folder `dist/` untuk Linux, Windows, dan macOS.

### Portabilitas Binary

Binary Linux dibuat dengan **static linking** untuk memastikan kompatibilitas di berbagai distribusi Linux tanpa masalah dependensi GLIBC. Binary ini dapat berjalan di sistem Linux dengan versi GLIBC yang berbeda.

## Instalasi Global

### Linux/macOS

```bash
cd data-splitter
./install.sh
```

Untuk uninstall:

```bash
cd data-splitter
./uninstall.sh
```

### Windows

Untuk Windows, ikuti panduan di `WINDOWS_INSTALL.md` atau gunakan script yang tersedia:

```powershell
# Install (PowerShell recommended)
.\install-windows.ps1 -BinaryPath "path\to\data-splitter-windows-amd64.exe"

# Uninstall
.\uninstall-windows.ps1
```

```cmd
# Install (batch, no admin required)
install-windows.bat "path\to\data-splitter-windows-amd64.exe"

# Uninstall
uninstall-windows.bat
```

## Konfigurasi (`config.yaml`)

Contoh file `config.yaml`:

```yaml
version: "1.0"

database:
  type: "${DB_TYPE}"
  host: "${DB_HOST}"
  port: ${DB_PORT}
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  source_db: "${DB_SOURCE_DB}"

tables:
  - name: "mockup_user_document"
    enabled: true
    split_column: "created_at"
    archive_pattern: "artywiz_{year}"

archive:
  years:
    - 2025
  options:
    batch_size: 100
    delete_after_archive: false
    create_archive_db: true
    dry_run: false

processing:
  log_level: "info"
  log_path: "logs/data-splitter.log"
  continue_on_error: true
  heartbeat_batch_interval: 10
```

**Catatan:**
- `config.yaml` dapat menggunakan placeholder `${VAR}` yang akan diisi dari environment variable
- Ini memungkinkan config dapat di-commit tanpa menyertakan secret

## Menyimpan Secret Lokal (.env)

Untuk pengembangan lokal, buat file `.env`:

```
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3307
DB_USER=root
DB_PASSWORD=changeme
DB_SOURCE_DB=artywiz
```

Loader akan memuat `.env` menggunakan `godotenv` dan mengisi placeholder dalam `config.yaml`.

## Cara Menjalankan

### Local Development

```bash
cd data-splitter
./data-splitter --config config.yaml
```

### CI/CD Pipeline

```bash
CI=true ./data-splitter --config config.yaml >stdout.log 2>stderr.log
```

### Lihat Informasi Direktori

```bash
./data-splitter --info
```

Output:
```
working_dir: /path/to/current/directory
project_dir: /path/to/data-splitter/source
```

## Output untuk Pipeline

Tool mencetak baris ringkas ke stdout yang mudah diparse oleh pipeline:

```
PROGRESS table=users year=2025 processed=0 total=0 batch=0 status=started
PROGRESS table=users year=2025 processed=1000 total=96716 batch=10
PROGRESS table=users year=2025 processed=96716 total=96716 batch=97 status=completed duration=1h23m45s
FINAL table=users year=2025 processed=96716 duration=1h23m45s exit=0
```

## Flag yang Tersedia

- `--config`: Path ke file konfigurasi (default: config.yaml)
- `--info`: Tampilkan informasi direktori working dan project

## Environment Variables

- `CI=true`: Disable UI untuk environment CI
- `NO_SPINNER=1`: Disable spinner
- `NO_PROGRESS=1`: Disable progress bar
- `LOG_LEVEL`: Set log level (debug/info/warn/error)
- `LOG_TAIL_LINES`: Jumlah baris log yang ditampilkan saat error

## Troubleshooting

- Jika melihat SQL di stdout, pastikan `setupLogging` sudah benar
- Pastikan tidak ada kode yang menulis log ke stdout secara eksplisit
- Untuk secret multi-line, gunakan file mounts

## Perubahan Penting

- PROGRESS/FINAL output untuk pipeline parsing
- Spinner ke stderr untuk menghindari interferensi dengan stdout
- Support LOG_TAIL_LINES untuk debugging error
- HEARTBEAT_BATCH_INTERVAL yang dapat dikonfigurasi
- Support .env via godotenv
- Flag --info untuk informasi direktori
- Flag --config untuk override path konfigurasi
- Cross-platform build support
- Windows installation scripts

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
# Data Splitter

Panduan singkat (Bahasa Indonesia) untuk membangun dan menjalankan tool
`data-splitter`. README ini berfokus pada penggunaan praktis, konfigurasi,
penanganan secret lokal (`.env`), dan output yang aman untuk pipeline.

## Ringkasan

- Tool ini memindahkan data dari tabel sumber ke database arsip per-tahun.
- Menyediakan batch processing, resume, dan fallback insert untuk kasus error
- Mengeluarkan baris PROGRESS/FINAL yang machine-friendly ke stdout agar CI dapat
  memantau progres tanpa terpengaruh oleh UI interaktif.

## Build

### Linux/macOS

Pastikan Go toolchain terpasang lalu di folder project jalankan:

```bash
cd data-splitter
go build -ldflags "-X main.projectDir=$(pwd)" -o data-splitter ./cmd
```

### Windows

Untuk Windows, gunakan cross-compilation dari Linux/macOS:

```bash
cd data-splitter
./build-cross-platform.sh
```

Atau build langsung di Windows dengan Go:

```cmd
cd data-splitter
go build -ldflags "-X main.projectDir=%cd%" -o data-splitter.exe .\cmd
```

### Cross-platform Build

Script `build-cross-platform.sh` akan membuat binary untuk semua platform:

```bash
./build-cross-platform.sh
```

Binary akan tersedia di folder `dist/` untuk Linux, Windows, dan macOS.

Binary akan terbentuk di folder saat ini dengan informasi project directory yang embedded.

## Instalasi Global

### Linux/macOS

Untuk menginstall data-splitter secara global agar bisa dijalankan dari mana saja:

```bash
cd data-splitter
./install.sh
```

Untuk uninstall:

```bash
cd data-splitter
./uninstall.sh
```

Atau manual:

```bash
# Build dengan project directory embedded
go build -ldflags "-X main.projectDir=$(pwd)" -o data-splitter ./cmd

# Install ke /usr/local/bin
sudo cp data-splitter /usr/local/bin/
sudo chmod +x /usr/local/bin/data-splitter
```

### Windows

Untuk Windows, ikuti panduan di `WINDOWS_INSTALL.md` atau gunakan script yang tersedia:

```powershell
# Install (PowerShell recommended)
.\install-windows.ps1 -BinaryPath "path\to\data-splitter-windows-amd64.exe"

# Uninstall
.\uninstall-windows.ps1
```

```cmd
# Install (batch, no admin required)
install-windows.bat "path\to\data-splitter-windows-amd64.exe"

# Uninstall
uninstall-windows.bat
```

Setelah install, Anda bisa menjalankan `data-splitter` dari direktori mana saja.

## Konfigurasi (`config.yaml`)

Contoh file `config.yaml` (sesuaikan):

```yaml
version: "1.0"

database:
  type: "${DB_TYPE}"
  host: "${DB_HOST}"
  port: ${DB_PORT}
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  source_db: "${DB_SOURCE_DB}"

tables:
  - name: "mockup_user_document"
    enabled: true
    split_column: "created_at"
    archive_pattern: "artywiz_{year}"

archive:
  years:
    - 2025
  options:
    batch_size: 100
    delete_after_archive: false
    create_archive_db: true
    dry_run: false

processing:
  log_level: "info"
  log_path: "logs/data-splitter.log"
  continue_on_error: true
  heartbeat_batch_interval: 10
```

Catatan:
- `config.yaml` dapat mengandung placeholder `${VAR}` yang akan diisi dari
  environment saat aplikasi dijalankan. Ini memungkinkan config dapat di-commit
  tanpa menyertakan secret.

## Menyimpan secret lokal (.env)

Untuk pengembangan lokal, buat file `.env` (pastikan tidak di-commit):

```
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3307
DB_USER=root
DB_PASSWORD=changeme
DB_SOURCE_DB=artywiz
```

Loader akan memuat `.env` (pakai `godotenv`) dan mengisi placeholder `${...}`
dalam `config.yaml`. Jika CI/production sudah menyediakan env vars, nilai CI
akan lebih prioritas dan tidak akan ditimpa oleh `.env`.

Pastikan `data-splitter/.gitignore` berisi `.env` dan `logs/` (sudah ditambahkan).

## Output untuk pipeline (PROGRESS/FINAL)

Tool mencetak baris ringkas ke stdout yang mudah diparse oleh pipeline. Contoh:

```
PROGRESS table=users year=2025 processed=0 total=0 batch=0 status=started
PROGRESS table=users year=2025 processed=1000 total=96716 batch=10
... (periodic heartbeat)
PROGRESS table=users year=2025 processed=96716 total=96716 batch=97 status=completed duration=1h23m45s
FINAL table=users year=2025 processed=96716 duration=1h23m45s exit=0
```

- `PROGRESS` dicetak periodik setiap N batch (konfigurasi `heartbeat_batch_interval`).
- Spinner/interactive UI ditulis ke stderr agar tidak mengganggu stdout yang dibaca
  oleh pipeline.

Jika terjadi error fatal, program akan mencetak tail dari log file ke stderr dan
menuliskan `FATAL: ...` lalu keluar dengan exit code non-zero.

## LOG_TAIL_LINES

Saat terjadi error kritis, tool akan mencetak sejumlah baris terakhir dari file log
(`processing.log_path`) ke stderr untuk membantu debugging. Jumlah baris bisa
diatur lewat env `LOG_TAIL_LINES` (opsional).

## HEARTBEAT_BATCH_INTERVAL

`heartbeat_batch_interval` menentukan frekuensi heartbeat (berapa batch antara
PROGRESS lines). Default 10. Pilih nilai yang sesuai berdasarkan `batch_size` dan
besar total data:

- Nilai kecil (1-5): visibilitas tinggi, tapi banyak output.
- Nilai sedang (10): seimbang.
- Nilai besar (50-100): output lebih sedikit, visibilitas lebih jarang.

## Cara Menjalankan

Local (menggunakan .env):

```bash
cd data-splitter
./data-splitter --config config.yaml
# atau lihat informasi direktori
./data-splitter --info
```

CI (non-interactive):

```bash
CI=true ./data-splitter --config config.yaml >stdout.log 2>stderr.log
```

Periksa `logs/data-splitter.log` untuk full structured logs.

## Flag --info

Flag `--info` menampilkan informasi direktori kerja dan direktori proyek:

```bash
./data-splitter --info
```

Output:
```
working_dir: /path/to/current/directory
project_dir: /path/to/data-splitter/source
```

- `working_dir`: Direktori tempat Anda menjalankan perintah data-splitter
- `project_dir`: Lokasi source code data-splitter (di-set saat build)

Berguna untuk memastikan tool memahami konteks direktori saat dijalankan secara global.

## Troubleshooting singkat

- Jika kamu melihat SQL atau pesan besar di stdout — pastikan `LOG_LEVEL` dan
  `setupLogging` sudah benar dan `config.processing.log_path` mengarahkan ke file.
- Untuk multi-line secret (mis. PEM private key) gunakan file-mounts/secret
  mounts dan referensikan path di config (mis. `ssl_key_file: "/run/secrets/key.pem"`).

## Rekap perubahan penting

- PROGRESS/FINAL one-liners untuk pipeline-safe parsing
- Spinner ke stderr
- `LOG_TAIL_LINES` support untuk tail-on-error
- `HEARTBEAT_BATCH_INTERVAL` configurable
- `.env` support via godotenv (tidak menimpa CI env)
- `--info` flag untuk menampilkan working_dir dan project_dir
- `--config` flag untuk override path konfigurasi
