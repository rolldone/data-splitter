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

Pastikan Go toolchain terpasang lalu di folder project jalankan:

```bash
cd data-splitter
go build ./...
```

Binary akan terbentuk di folder saat ini.

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
```

CI (non-interactive):

```bash
CI=true ./data-splitter --config config.yaml >stdout.log 2>stderr.log
```

Periksa `logs/data-splitter.log` untuk full structured logs.

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

---

Jika mau, saya bisa menambahkan skrip `scripts/verify-ui.sh` untuk memvalidasi
perilaku interactive vs CI, atau menambahkan helper untuk membaca secret file
otomatis dari config (file-ref). Beritahu saya mana yang diinginkan.