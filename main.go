package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

// ─── Request / Response Structs ──────────────────────────────────────────────

// StudentPayload represents the incoming JSON request body.
type StudentPayload struct {
	StudentName      string `json:"student_name"       binding:"required"`
	Age              int    `json:"age"                binding:"required"`
	ChildObservation string `json:"child_observation"  binding:"required"`
	TeacherNotes     string `json:"teacher_notes"      binding:"required"`
	ParentHopes      string `json:"parent_hopes"       binding:"required"`
}

// InsightResponse is the structured JSON response returned by the API.
type InsightResponse struct {
	TalentLabel            string   `json:"talent_label"`
	PersonalityAnalysis    string   `json:"personality_analysis"`
	ParentRecommendations  []string `json:"parent_recommendations"`
	TeacherRecommendations []string `json:"teacher_recommendations"`
	Sources                []string `json:"sources"` // nama file PDF sumber konteks RAG
}

// ErrorResponse wraps error messages in a consistent JSON format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ─── Globals ──────────────────────────────────────────────────────────────────

var (
	geminiLLM  *googleai.GoogleAI
	chromaURL  string
	geminiKey  string
)

// ─── main ────────────────────────────────────────────────────────────────────

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	geminiKey = os.Getenv("GOOGLE_API_KEY")
	if geminiKey == "" {
		log.Fatal("FATAL: GOOGLE_API_KEY is not set")
	}

	chromaURL = os.Getenv("CHROMADB_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Gemini LLM
	ctx := context.Background()
	var err error
	geminiLLM, err = googleai.New(ctx,
		googleai.WithAPIKey(geminiKey),
		googleai.WithDefaultModel("gemini-2.5-flash"),
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Gemini: %v", err)
	}

	// Set up Gin router
	router := gin.Default()

	// CORS — allow all origins for MVP
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// ── Routes ────────────────────────────────────────────────────────────────

	router.GET("/health", healthHandler)
	router.POST("/api/generate-insights", generateInsightsHandler)

	log.Printf("🚀 Bimbi AI Backend started on http://localhost:%s", port)
	log.Printf("📡 ChromaDB: %s", chromaURL)
	log.Printf("✅ POST /api/generate-insights ready")

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("FATAL: Server failed: %v", err)
	}
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "bimbi-ai-backend",
		"version": "1.0.0",
	})
}

func generateInsightsHandler(c *gin.Context) {
	var payload StudentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_payload",
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	ctx := c.Request.Context()

	// Build query string for semantic search
	queryText := buildQueryText(payload)

	// Retrieve top-3 relevant chunks from ChromaDB via HTTP
	ragContext, sources, err := queryChromaDB(ctx, queryText, 3)
	if err != nil {
		log.Printf("Warning: ChromaDB query failed: %v — proceeding without RAG context", err)
		ragContext = "Tidak ada konteks knowledge base eksternal. Gunakan pengetahuan ahli Anda."
		sources = []string{}
	}

	// Build the full prompt
	fullPrompt := buildPrompt(ragContext, payload)

	// Call Gemini LLM
	rawResponse, err := geminiLLM.Call(ctx, fullPrompt,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(8192),
	)
	if err != nil {
		log.Printf("ERROR: LLM call failed: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "llm_error",
			Message: "Failed to generate insights from AI model",
		})
		return
	}

	// Parse structured JSON from LLM response
	insight, err := parseInsightJSON(rawResponse)
	if err != nil {
		log.Printf("ERROR: JSON parse failed: %v\nRaw: %s", err, rawResponse)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "parse_error",
			Message: "AI model returned an unexpected format. Please retry.",
		})
		return
	}

	// Inject sumber dokumen RAG ke response (dari metadata ChromaDB, bukan dari LLM)
	insight.Sources = sources

	c.JSON(http.StatusOK, insight)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildQueryText(p StudentPayload) string {
	return fmt.Sprintf(
		"Child %s, age %d. Observations: %s. Teacher notes: %s. Parent hopes: %s",
		p.StudentName, p.Age, p.ChildObservation, p.TeacherNotes, p.ParentHopes,
	)
}

func buildPrompt(ragContext string, p StudentPayload) string {
	system := `Anda adalah seorang psikolog pendidikan dan spesialis perkembangan anak berpengalaman lebih dari 20 tahun.
Analisis profil anak menggunakan konteks riset yang disediakan (dari teori Multiple Intelligences Howard Gardner dan psikologi pendidikan modern).
Bersikaplah mendorong, berbasis bukti, spesifik, dan peka budaya.

🇮🇩 WAJIB: SELURUH respons HARUS dalam Bahasa Indonesia. Tidak ada satu kata pun dalam bahasa Inggris atau bahasa lain yang diizinkan dalam nilai-nilai JSON.
Ini adalah guardrail yang tidak boleh dilanggar. Pengguna adalah orang tua dan guru Indonesia.
Pastikan jawaban Anda padat, jelas, dan ringkas untuk menghindari pemotongan (truncation).

CRITICAL: Balas HANYA dengan objek JSON yang valid. Tidak ada markdown, tidak ada code fence, tidak ada penjelasan di luar JSON.
Struktur JSON yang diperlukan:
{
  "talent_label": "<label bakat utama dalam Bahasa Indonesia, misal: 'Kecerdasan Spasial-Visual', 'Kecerdasan Musikal-Ritmis', 'Kecerdasan Kinestetik-Jasmani', 'Kecerdasan Linguistik-Verbal', 'Kecerdasan Logis-Matematis', 'Kecerdasan Interpersonal', 'Kecerdasan Intrapersonal', 'Kecerdasan Naturalis'>",
  "personality_analysis": "<Dua paragraf lengkap dalam Bahasa Indonesia. Paragraf pertama: uraikan sifat kepribadian dan kekuatan kognitif anak berdasarkan observasi. Paragraf kedua: uraikan gaya belajar optimal anak dan bagaimana lingkungan membentuk perkembangannya.>",
  "parent_recommendations": [
    "<Rekomendasi spesifik dan dapat ditindaklanjuti untuk orang tua, dalam Bahasa Indonesia>",
    "<Rekomendasi spesifik dan dapat ditindaklanjuti untuk orang tua, dalam Bahasa Indonesia>",
    "<Rekomendasi spesifik dan dapat ditindaklanjuti untuk orang tua, dalam Bahasa Indonesia>"
  ],
  "teacher_recommendations": [
    "<Strategi kelas yang spesifik untuk guru, dalam Bahasa Indonesia>",
    "<Strategi kelas yang spesifik untuk guru, dalam Bahasa Indonesia>"
  ]
}

INGAT: Semua nilai string di atas HARUS ditulis dalam Bahasa Indonesia yang baik dan benar.`

	user := fmt.Sprintf(`## Konteks Riset Psikologi Pendidikan (RAG):
%s

---

## Profil Murid:
- **Nama:** %s
- **Usia:** %d tahun
- **Observasi Perilaku (Orang Tua/Pengamat):** %s
- **Catatan Guru:** %s
- **Harapan & Tujuan Orang Tua:** %s

Analisis profil ini dan kembalikan HANYA objek JSON dalam Bahasa Indonesia.`,
		ragContext,
		p.StudentName,
		p.Age,
		p.ChildObservation,
		p.TeacherNotes,
		p.ParentHopes,
	)

	return system + "\n\n" + user
}

func parseInsightJSON(raw string) (*InsightResponse, error) {
	// Strip markdown code fences if present
	cleaned := strings.TrimSpace(raw)
	for _, fence := range []string{"```json", "```"} {
		cleaned = strings.TrimPrefix(cleaned, fence)
	}
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// Extract JSON object boundaries
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON object found in LLM response")
	}
	jsonStr := cleaned[start : end+1]

	var insight InsightResponse
	if err := json.Unmarshal([]byte(jsonStr), &insight); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	return &insight, nil
}
