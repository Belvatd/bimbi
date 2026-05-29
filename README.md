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

# Cek status container (pastikan postgres dan chromadb berjalan)
docker ps
```

### 3. Siapkan & Ingest Dokumen RAG

```bash
# Taruh file referensi PDF/TXT di folder ./source_documents/

# Jalankan proses ingestion
go run cmd/ingester/main.go
```

### 4. Jalankan API Server

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
Menghasilkan *insight* psikologi menggunakan RAG.
**Header:** `Authorization: Bearer <token_dari_login>`

**Request Body:**
```json
{
  "student_name": "Budi Santoso",
  "age": 8,
  "child_observation": "Suka menggambar, membangun balok, dan sering bertanya tentang bagaimana sesuatu bekerja",
  "teacher_notes": "Sangat fokus saat kegiatan seni dan konstruksi. Mudah bosan dengan tugas menulis.",
  "parent_hopes": "Ingin anak berkembang di bidang yang sesuai bakatnya dan percaya diri"
}
```

**Response:**
```json
{
  "talent_label": "Kecerdasan Spasial-Visual",
  "personality_analysis": "Budi menunjukkan kecerdasan spasial-visual yang kuat...\n\nDalam konteks pembelajaran...",
  "parent_recommendations": [
    "Sediakan kit konstruksi seperti LEGO atau Minecraft untuk eksplorasi",
    "Kunjungi museum sains atau pameran seni secara rutin",
    "Dukung hobi menggambar dengan menyediakan alat gambar berkualitas"
  ],
  "teacher_recommendations": [
    "Gunakan diagram dan peta konsep visual dalam pembelajaran",
    "Berikan tugas berbasis proyek seperti membuat model 3D atau presentasi visual"
  ],
  "sources": [
    "buku_panduan_bakat.pdf"
  ]
}
```

---

## ⚙️ Tech Stack

| Komponen | Teknologi |
|---|---|
| Bahasa | Go 1.21+ |
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
| `CHROMADB_URL` | URL ChromaDB lokal | `http://localhost:8000` |
| `DB_HOST` | Host PostgreSQL | `localhost` |
| `DB_PORT` | Port PostgreSQL | `5432` |
| `DB_USER` | Username PostgreSQL | **(Wajib diisi)** |
| `DB_PASSWORD` | Password PostgreSQL | **(Wajib diisi)** |
| `DB_NAME` | Nama database PostgreSQL | **(Wajib diisi)** |
| `JWT_SECRET` | Kunci rahasia untuk enkripsi token JWT | **(Wajib diisi)** |
