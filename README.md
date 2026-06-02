# 🧠 Bimbi AI Backend

Backend MVP untuk aplikasi EdTech **Bimbi AI** — mendeteksi bakat tersembunyi anak dan memberikan rekomendasi pembelajaran personal menggunakan arsitektur **Fully RAG (Retrieval-Augmented Generation)**. Sistem ini dibangun menggunakan praktik terbaik **Clean Architecture**.

> **Tidak ada ML tradisional.** Semua analisis dilakukan oleh LLM (Gemini 2.5 Flash) yang diperkaya dengan konteks dari knowledge base (ChromaDB).

---

## 🏗️ Arsitektur

### Clean Architecture

Proyek ini telah direfaktor untuk mematuhi konsep **Clean Architecture**, sehingga pemisahan kekhawatiran (separation of concerns) menjadi sangat jelas:

1.  **Domain Layer (`internal/domain`)**: Inti dari sistem. Hanya berisi definisi struktur data dan antarmuka (*interface*). Tidak bergantung pada kerangka kerja luar.
2.  **Repository Layer (`internal/repository`)**: Implementasi dari akses data, seperti PostgreSQL (menggunakan GORM) dan LLM/VectorDB (Chroma & Gemini).
3.  **Service Layer (`internal/service`)**: Tempat di mana logika bisnis utama berada, seperti algoritma pembuatan *prompt* RAG dan pembuatan token JWT.
4.  **Handler Layer (`internal/handler` & `middleware`)**: Lapisan transportasi HTTP yang dikelola menggunakan Gin. Menangani permintaan dan respons JSON.

### Alur Kerja RAG

```
Frontend Request
      │
      ▼
┌──────────────────┐
│ Gin API          │  POST /api/generate-insights (Dilindungi JWT)
│ (cmd/api)        │
└────────┬─────────┘
         │  1. Semantic Search
         ▼
┌──────────────────┐
│ ChromaDB         │  Vector Store (Docker)
│ (localhost:8000) │  ← Diisi oleh cmd/ingester
└────────┬─────────┘
         │  2. Top-3 RAG Context
         ▼
┌──────────────────┐
│ Gemini 2.5 Flash │  LLM Reasoning
│ (Google AI)      │  + RAG Prompt
└────────┬─────────┘
         │  3. Structured JSON
         ▼
    JSON Response
```

---

## 📁 Struktur Proyek

```
bimbi/
├── cmd/
│   ├── api/                 # Entry point untuk API Server (Gin, Auth, RAG)
│   │   └── main.go
│   └── ingester/            # Entry point untuk skrip Ingestion Dokumen
│       └── main.go
├── internal/
│   ├── config/              # Pemuatan environment (.env)
│   ├── domain/              # Antarmuka dan entitas (Student, User, dsb)
│   ├── handler/             # HTTP Controllers (Auth & Insight)
│   ├── middleware/          # Middleware (JWT Auth)
│   ├── repository/          # GORM Postgres, ChromaDB, dan Gemini repo
│   └── service/             # Logika bisnis RAG dan Auth
├── source_documents/        # Taruh file PDF psikologi pendidikan di sini
├── docker-compose.yml       # Konfigurasi container ChromaDB & PostgreSQL
├── .env                     # Kredensial & konfigurasi
└── go.mod / go.sum
```

---

## 🚀 Cara Menjalankan

### 1. Siapkan Environment

```bash
# Isi variabel di file .env
nano .env

# Variabel yang harus ada (lihat bagian Environment Variables)
```

### 2. Jalankan Database (PostgreSQL & ChromaDB)

```bash
# Pastikan Docker Desktop sudah aktif
docker-compose up -d

# Cek status container (pastikan postgres, chromadb, mathesar, dan mathesar-db berjalan)
docker ps
```

### 3. Mathesar Database Viewer (Optional)
Kami telah menyertakan **Mathesar**, GUI database berbasis web, agar Anda dapat dengan mudah menginspeksi tabel database PostgreSQL (`users`, `students`, dll).

1. Buka browser dan akses **[http://localhost:8000](http://localhost:8000)**.
2. Buat akun admin baru saat pertama kali setup.
3. Pilih **Add Database** dan hubungkan ke database Bimbi dengan detail berikut:
   * **Database Engine:** `PostgreSQL`
   * **Host:** `postgres` (koneksi internal sesama container docker)
   * **Port:** `5432`
   * **Database Name:** `bimbi_db` (sesuai nilai `DB_NAME` di `.env`)
   * **Username:** `bimbi` (sesuai nilai `DB_USER` di `.env`)
   * **Password:** `bimbi_password` (sesuai nilai `DB_PASSWORD` di `.env`)

### 4. Siapkan & Ingest Dokumen RAG

```bash
# Taruh file referensi PDF/TXT di folder ./source_documents/

# Jalankan proses ingestion
go run cmd/ingester/main.go
```

### 5. Jalankan API Server

```bash
go run cmd/api/main.go
# Server berjalan di: http://localhost:8080
```

---

## 📡 API Endpoints

### Kumpulan Endpoint Auth (Tidak dilindungi)

#### `POST /api/auth/register`
Mendaftarkan pengguna baru ke database PostgreSQL.
**Request:**
```json
{
  "email": "guru@bimbi.id",
  "password": "password123"
}
```

#### `POST /api/auth/login`
Melakukan otentikasi dan mengembalikan Bearer Token (JWT).
**Response:**
```json
{
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5c..."
}
```

---

### Kumpulan Endpoint Insight (Dilindungi JWT)

#### `POST /api/generate-insights`
Menghasilkan *insight* psikologi perkembangan anak untuk orang tua menggunakan RAG.
**Header:** `Authorization: Bearer <token_dari_login>`

**Request Body:**
```json
{
  "child_name": "Ara",
  "child_age": 5,
  "daily_activities": [
    "menggambar",
    "bermain balok",
    "menonton video edukasi"
  ],
  "parent_anxiety": "Anak saya susah fokus dan selalu bergerak, saya khawatir dia tidak bisa belajar dengan baik di sekolah.",
  "positive_triggers": "Sangat antusias saat diajak membangun sesuatu dari balok atau kardus bekas.",
  "parent_goals": "Ingin tahu cara terbaik mendampingi anak belajar di rumah sesuai karakternya."
}
```

**Response:**
```json
{
  "talent_label": "Kecerdasan Spasial-Visual",
  "empathetic_analysis": "Kekhawatiran Anda sangat wajar dan banyak orang tua merasakannya. Anak yang terus bergerak seringkali bukan tanda masalah, melainkan tanda energi kognitif yang tinggi.\n\nBerdasarkan aktivitas seperti membangun balok dan menggambar, Ara menunjukkan kecerdasan Spasial-Visual yang kuat. Kemampuan ini adalah fondasi bagi profesi seperti arsitek, desainer, dan insinyur.",
  "home_activities": [
    "Bangun 'kota mini' dari kardus bekas sabun dan botol shampoo bersama Ara selama 15 menit sebelum tidur.",
    "Ajak Ara mendekorasi tempat belajarnya sendiri dengan gambar-gambar yang dia buat.",
    "Sediakan plastisin atau tanah liat untuk membuat bentuk hewan favoritnya."
  ],
  "learning_hacks": [
    "Saat menjelaskan sesuatu, gunakan gambar atau sketsa sederhana, bukan hanya kata-kata.",
    "Beri Ara 'tugas visual': minta dia menggambar apa yang baru dipelajari sebagai pengganti menulis ringkasan."
  ],
  "sources": [
    "multiple_intelligences_gardner.pdf"
  ]
}
```

---

## 🐋 Manual Docker Ingest Job

Untuk melakukan ingestion dokumen di server atau lingkungan deployment (seperti Portainer):

### Menggunakan CLI/Docker Compose
Jika menggunakan docker-compose di VM, Anda dapat men-trigger job ini dengan perintah:
```bash
docker compose run --rm ingester
```

### Menggunakan Portainer UI
1. Buka Portainer -> **Containers** -> Klik **Add Container**.
2. Masukkan konfigurasinya:
   * **Name**: `bimbi-ingester-manual`
   * **Image**: Gunakan image backend Anda (misal: `bimbi-backend:latest`).
   * **Command**: `[ "./ingester" ]` (atau tulis `./ingester` pada override command).
   * **Env**: Tambahkan `GOOGLE_API_KEY` dan `CHROMADB_URL=http://chromadb:8000` (atau port internal `8001` sesuai compose).
   * **Network**: Pilih network yang sama dengan stack Anda (misal: `<nama_stack>_default`).
3. Klik **Deploy the container**. Container akan memproses file PDF lalu otomatis mati setelah selesai (`Stopped`).

---

## ⚙️ Tech Stack

| Komponen | Teknologi |
|---|---|
| Bahasa | Go 1.22+ |
| Web Framework | Gin (gin-gonic) |
| Relational DB | PostgreSQL (via GORM) |
| Vector Store | ChromaDB (via Docker) |
| AI / LLM | Google Gemini 2.5 Flash |
| Security | JWT (JSON Web Tokens), bcrypt |
| Env Management | godotenv |

## 🔑 Environment Variables

Pastikan file `.env` di direktori utama (root) berisi konfigurasi berikut:

| Variable | Deskripsi | Default |
|---|---|---|
| `PORT` | Port untuk API Server | `8080` |
| `GOOGLE_API_KEY` | API Key dari Google AI Studio | **(Wajib diisi)** |
| `CHROMADB_URL` | URL ChromaDB lokal | `http://localhost:8001` |
| `DB_HOST` | Host PostgreSQL | `localhost` |
| `DB_PORT` | Port PostgreSQL | `5432` |
| `DB_USER` | Username PostgreSQL | **(Wajib diisi)** |
| `DB_PASSWORD` | Password PostgreSQL | **(Wajib diisi)** |
| `DB_NAME` | Nama database PostgreSQL | **(Wajib diisi)** |
| `JWT_SECRET` | Kunci rahasia untuk enkripsi token JWT | **(Wajib diisi)** |
