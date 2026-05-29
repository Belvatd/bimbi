package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bimbi-backend/internal/domain"
)

type ragService struct {
	vectorRepo domain.VectorRepo
	llmRepo    domain.LLMRepo
}

func NewRagService(vectorRepo domain.VectorRepo, llmRepo domain.LLMRepo) domain.RagService {
	return &ragService{
		vectorRepo: vectorRepo,
		llmRepo:    llmRepo,
	}
}

func (s *ragService) GenerateInsights(ctx context.Context, payload domain.StudentPayload) (*domain.InsightResponse, error) {
	queryText := s.buildQueryText(payload)

	ragContext, sources, err := s.vectorRepo.Query(ctx, queryText, 10)
	if err != nil {
		ragContext = "Tidak ada konteks knowledge base eksternal. Gunakan pengetahuan ahli Anda."
		sources = []string{}
	}

	fullPrompt := s.buildPrompt(ragContext, payload)

	rawResponse, err := s.llmRepo.Call(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm error: %w", err)
	}

	insight, err := s.parseInsightJSON(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	insight.Sources = sources
	return insight, nil
}

func (s *ragService) buildQueryText(p domain.StudentPayload) string {
	return fmt.Sprintf(
		"Child %s, age %d. Observations: %s. Teacher notes: %s. Parent hopes: %s",
		p.StudentName, p.Age, p.ChildObservation, p.TeacherNotes, p.ParentHopes,
	)
}

func (s *ragService) buildPrompt(ragContext string, p domain.StudentPayload) string {
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

func (s *ragService) parseInsightJSON(raw string) (*domain.InsightResponse, error) {
	cleaned := strings.TrimSpace(raw)
	for _, fence := range []string{"```json", "```"} {
		cleaned = strings.TrimPrefix(cleaned, fence)
	}
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON object found in LLM response")
	}
	jsonStr := cleaned[start : end+1]

	var insight domain.InsightResponse
	if err := json.Unmarshal([]byte(jsonStr), &insight); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	return &insight, nil
}
