# 🌍 Gozaq Environment Profiles (`env/`)

Folder ini menyediakan template variabel lingkungan (*environment variables*) yang sudah dioptimasi untuk berbagai mode *deployment*:

---

## 📁 File Profiles

| File | Keterangan | Target Environment |
| :--- | :--- | :--- |
| **[`.env.development`](.env.development)** | Konfigurasi lokal dev. Menggunakan `LogMailer` (stdout preview), database lokal tanpa SSL, dan durasi timeout fleksibel. | Local Development (`make dev`) |
| **[`.env.production`](.env.production)** | Konfigurasi production siap pakai dengan pool database diperbesar, SSL/TLS enforcement, SMTP provider nyata, dan JWT expiry pendek (60m). | Staging / Production Server |
| **[`.env.example`](.env.example)** | Master template lengkap dengan penjelasan setiap variabel. | Reference Template |

---

## 🚀 Cara Menggunakan

### 1. Untuk Mode Development:
```bash
# Salin profile dev ke root .env
cp env/.env.development .env

# Jalankan server
make dev
```

### 2. Untuk Mode Production:
```bash
# Salin profile prod ke root .env
cp env/.env.production .env

# Sesuaikan kredensial DATABASE_URL, REDIS_URL, JWT_SECRET, dan SMTP
nano .env

# Build & run
make build
./bin/api
```
