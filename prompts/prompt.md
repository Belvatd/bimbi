# Prompt for AI Agent: Bimbi AI Backend (Go + RAG)

**Role:** You are an expert Backend Go Developer and AI Engineer. 
**Task:** I want to build the backend MVP for an EdTech application named 'Bimbi AI' using Golang. The app detects a child's hidden talents and provides personalized learning recommendations using a Fully RAG (Retrieval-Augmented Generation) architecture. There is NO traditional Machine Learning; everything relies on RAG and LLM reasoning.

## Tech Stack
* **Language:** Go (Golang) 1.21+
* **Web Framework:** `github.com/gin-gonic/gin` (for API)
* **AI/RAG Framework:** `github.com/tmc/langchaingo` (for LLM orchestration and embeddings)
* **LLM Provider:** Google Gemini (Gemini 1.5 Flash via langchaingo's googleai)
* **Vector Store:** ChromaDB (Running locally via Docker)
* **Environment Variables:** `github.com/joho/godotenv`

## Project Architecture & Tasks
Please outline the directory structure and write the complete, production-ready code for this project.

### Step 1: Infrastructure & Initialization
1. Provide the exact terminal commands to initialize the module (`go mod init bimbi-backend`) and fetch dependencies.
2. Create a `docker-compose.yml` file at the root to run `chromadb/chroma`. Expose it on port `8000:8000` and map a local volume `./chroma_data:/chroma/chroma` so the vector data persists.
3. Create an empty folder named `/source_documents`.
4. Create a `.env` file containing `GOOGLE_API_KEY=your_key_here` and `CHROMADB_URL=http://localhost:8000`.

### Step 2: The Ingestion Script (`ingestion/ingest.go`)
1. Write a standalone Go script that reads PDF files from the `/source_documents` directory.
2. Split the extracted text into manageable chunks.
3. Generate embeddings using Gemini and store them into the ChromaDB instance running on `localhost:8000`. 

### Step 3: The Gin API Server (`main.go`)
1. Initialize a Gin router with CORS middleware to allow all origins.
2. Define a Go `struct` for the incoming JSON payload containing: `student_name` (string), `age` (int), `child_observation` (string), `teacher_notes` (string), and `parent_hopes` (string).
3. Create a `POST /api/generate-insights` endpoint.
4. Inside the endpoint, retrieve the top 3 most relevant contexts from the ChromaDB vector store based on the incoming payload.
5. Construct a system prompt instructing the LLM to act as an expert educational psychologist. It must use the retrieved RAG context and the student's payload to generate a highly accurate analysis.
6. Call the Gemini LLM and return a structured JSON response containing exactly these fields: `talent_label` (string), `personality_analysis` (string, 2 paragraphs), `parent_recommendations` (array of 3 strings), and `teacher_recommendations` (array of 2 strings).

*Note: Please output the exact folder structure and the complete code for `docker-compose.yml`, `main.go`, and `ingestion/ingest.go`. Do not use placeholder code for the RAG chain; write the actual implementation using `langchaingo`.*

---

## Urutan Eksekusi (Panduan Developer)

Setelah AI Agent selesai membuat struktur folder dan kode, ikuti langkah berikut di Terminal (macOS):

1. **Jalankan Database Vektor (ChromaDB)**
   * Pastikan Docker Desktop sudah menyala.
   * Jalankan perintah:
     ```bash
     docker-compose up -d
     ```
2. **Siapkan Data & Kredensial**
   * Buka file `.env` dan masukkan `GOOGLE_API_KEY` milik Anda.
   * Pindahkan file PDF referensi (seperti jurnal psikologi dan proposal) ke dalam folder `/source_documents`.
3. **Proses Dokumen (Ingestion)**
   * Masukkan data teks ke dalam vektor dengan perintah:
     ```bash
     go run ingestion/ingest.go
     ```
4. **Nyalakan Server API Utama**
   * Mulai server Gin dengan perintah:
     ```bash
     go run main.go
     ```
   * API siap menerima *request* di `http://localhost:8080/api/generate-insights`.