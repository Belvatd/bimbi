# 🧠 Bimbi AI Backend

Backend MVP untuk aplikasi EdTech **Bimbi AI** — mendeteksi bakat tersembunyi anak dan memberikan rekomendasi pembelajaran personal menggunakan arsitektur **Fully RAG (Retrieval-Augmented Generation)**.

> **Tidak ada ML tradisional.** Semua analisis dilakukan oleh LLM (Gemini 1.5 Flash) yang diperkaya dengan konteks dari knowledge base (ChromaDB).

---

## 🏗️ Arsitektur

```
Frontend Request
      │
      ▼
┌─────────────────┐
│   Gin API       │  POST /api/generate-insights
│   (main.go)     │
└────────┬────────┘
         │  1. Semantic Search
         ▼
┌─────────────────┐
│   ChromaDB      │  Vector Store (Docker)
│  (localhost:8000)│  ← Indexed by ingest.go
└────────┬────────┘
         │  2. Top-3 RAG Context
         ▼
┌─────────────────┐
│  Gemini 1.5     │  LLM Reasoning
│  Flash (Google) │  + RAG Prompt
└────────┬────────┘
         │  3. Structured JSON
         ▼
    JSON Response
```

## 📁 Struktur Proyek

```
bimbi/
├── main.go                 # Gin API server (entry point)
├── chroma_client.go        # ChromaDB HTTP client
├── ingestion/
│   └── ingest.go           # PDF ingestion & embedding script
├── source_documents/       # Taruh file PDF/TXT di sini
├── docker-compose.yml      # ChromaDB container
├── .env                    # API keys (TIDAK di-commit)
├── go.mod
└── go.sum
```

## 🚀 Cara Menjalankan

### 1. Siapkan Environment

```bash
# Clone / masuk ke direktori project
cd bimbi

# Isi API key di .env
nano .env
# Ubah: GOOGLE_API_KEY=your_actual_key_here
```

### 2. Jalankan ChromaDB (Vector Database)

```bash
# Pastikan Docker Desktop sudah aktif
docker-compose up -d

# Verifikasi berjalan
curl http://localhost:8000/api/v1/heartbeat
```

### 3. Siapkan & Ingest Dokumen

```bash
# Taruh file PDF psikologi pendidikan ke folder ini:
# ./source_documents/

# Jalankan ingestion
go run ingestion/ingest.go
```

### 4. Jalankan API Server

```bash
go run main.go
# Server berjalan di: http://localhost:8080
```

---

## 📡 API Endpoints

### `GET /health`
Cek status server.

**Response:**
```json
{
  "status": "ok",
  "service": "bimbi-ai-backend",
  "version": "1.0.0"
}
```

---

### `POST /api/generate-insights`

Analisis profil anak dan hasilkan rekomendasi bakat.

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
  "talent_label": "Spatial-Visual Intelligence",
  "personality_analysis": "Budi menunjukkan kecerdasan spasial-visual yang kuat...\n\nDalam konteks pembelajaran...",
  "parent_recommendations": [
    "Sediakan kit konstruksi seperti LEGO atau Minecraft untuk eksplorasi",
    "Kunjungi museum sains atau pameran seni secara rutin",
    "Dukung hobi menggambar dengan menyediakan alat gambar berkualitas"
  ],
  "teacher_recommendations": [
    "Gunakan diagram dan peta konsep visual dalam pembelajaran",
    "Berikan tugas berbasis proyek seperti membuat model 3D atau presentasi visual"
  ]
}
```

---

## ⚙️ Tech Stack

| Komponen | Teknologi |
|---|---|
| Language | Go 1.21+ |
| Web Framework | Gin (gin-gonic) |
| AI/LLM | Google Gemini 1.5 Flash |
| Embeddings | Google text-embedding-004 |
| RAG Orchestration | langchaingo |
| Vector Store | ChromaDB (Docker) |
| Env Management | godotenv |

## 🔑 Environment Variables

| Variable | Deskripsi |
|---|---|
| `GOOGLE_API_KEY` | API Key dari Google AI Studio |
| `CHROMADB_URL` | URL ChromaDB (default: `http://localhost:8000`) |
| `PORT` | Port server (default: `8080`) |
