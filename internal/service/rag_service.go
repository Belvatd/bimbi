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

func (s *ragService) GenerateInsights(ctx context.Context, payload domain.AssessmentRequest) (*domain.InsightResponse, error) {
	queryText := s.buildQueryText(payload)

	ragContext, chromaSources, err := s.vectorRepo.Query(ctx, queryText, 10)
	if err != nil {
		ragContext = "Tidak ada konteks knowledge base eksternal. Gunakan pengetahuan ahli Anda."
		chromaSources = []string{}
	}

	fullPrompt := s.buildPrompt(ragContext, chromaSources, payload)

	rawResponse, err := s.llmRepo.Call(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm error: %w", err)
	}

	insight, err := s.parseInsightJSON(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return insight, nil
}

// buildQueryText constructs the ChromaDB vector search string from
// the parent's daily activity observations and positive triggers.
func (s *ragService) buildQueryText(p domain.AssessmentRequest) string {
	activities := strings.Join(p.DailyActivities, ", ")
	return fmt.Sprintf(
		"Child likes: %s. Engaged when: %s.",
		activities,
		p.PositiveTriggers,
	)
}

// buildPrompt assembles the full system + user prompt for Gemini.
func (s *ragService) buildPrompt(ragContext string, chromaSources []string, p domain.AssessmentRequest) string {
	activities := strings.Join(p.DailyActivities, ", ")

	system := `Anda adalah psikolog anak dan pakar parenting modern yang sangat empatik.
Tugas Anda adalah membantu orang tua memahami potensi tersembunyi anak mereka berdasarkan observasi sehari-hari di rumah.

Nada bicara: Hangat, menenangkan, dan praktis. Hindari jargon akademis.

== ATURAN PENGGUNAAN KONTEKS (WAJIB DIPATUHI) ==
Anda akan menerima "Konteks Riset" yang diambil dari knowledge base internal (ChromaDB).
Setiap blok konteks diberi label "[Konteks N — Sumber: nama_file.pdf]" yang menunjukkan dari PDF mana teks tersebut berasal.
Anda HARUS menggunakan konteks tersebut sebagai sumber pengetahuan UTAMA dan PERTAMA.
Semua analisis talent_label, empathetic_analysis, home_activities, dan learning_hacks HARUS berakar dari teori dan fakta yang ada di dalam Konteks Riset tersebut.
Jika Konteks Riset memuat teori Multiple Intelligences atau gaya belajar tertentu, gunakan terminologi dan kerangka pikir dari sana.
Hanya gunakan pengetahuan umum Anda sebagai PELENGKAP jika konteks tidak mencakup aspek tertentu. Jangan pernah mengabaikan konteks yang diberikan.
Untuk field "sources": cantumkan HANYA nama file PDF (dari label "Sumber:") yang BENAR-BENAR Anda gunakan sebagai referensi dalam analisis ini. Jangan cantumkan PDF yang tidak Anda rujuk.

🇮🇩 WAJIB: SELURUH respons HARUS dalam Bahasa Indonesia. Tidak ada kata dalam bahasa lain.
Pastikan jawaban padat, jelas, dan ringkas agar tidak terpotong.

CRITICAL: Balas HANYA dengan objek JSON yang valid. Tidak ada markdown, tidak ada code fence, tidak ada penjelasan di luar JSON.
Struktur JSON yang diperlukan:
{
  "talent_label": "<label kecerdasan utama anak dalam Bahasa Indonesia — harus berdasarkan teori di Konteks Riset, misal: 'Kecerdasan Spasial-Visual', 'Kecerdasan Musikal-Ritmis', 'Kecerdasan Kinestetik-Jasmani', 'Kecerdasan Linguistik-Verbal', 'Kecerdasan Logis-Matematis', 'Kecerdasan Interpersonal', 'Kecerdasan Intrapersonal', 'Kecerdasan Naturalis'>",
  "empathetic_analysis": "<Dua paragraf dalam Bahasa Indonesia yang BERAKAR dari Konteks Riset. Paragraf pertama: validasi kekhawatiran orang tua dengan empati. Paragraf kedua: bingkai ulang kekhawatiran tersebut sebagai potensi bakat tersembunyi berdasarkan teori dari Konteks Riset, aktivitas harian, dan positive triggers anak.>",
  "home_activities": [
    "<Rekomendasi aktivitas rumah yang spesifik, terinspirasi dari Konteks Riset, menggunakan benda-benda rumah tangga biasa, dalam Bahasa Indonesia>",
    "<Rekomendasi aktivitas rumah yang spesifik, terinspirasi dari Konteks Riset, menggunakan benda-benda rumah tangga biasa, dalam Bahasa Indonesia>",
    "<Rekomendasi aktivitas rumah yang spesifik, terinspirasi dari Konteks Riset, menggunakan benda-benda rumah tangga biasa, dalam Bahasa Indonesia>"
  ],
  "learning_hacks": [
    "<Tips mendampingi gaya belajar anak di rumah — berdasarkan pendekatan dari Konteks Riset, dalam Bahasa Indonesia>",
    "<Tips mendampingi gaya belajar anak di rumah — berdasarkan pendekatan dari Konteks Riset, dalam Bahasa Indonesia>"
  ],
  "sources": ["<nama_file.pdf yang benar-benar Anda jadikan referensi>"]
}

INGAT: Semua nilai string HARUS dalam Bahasa Indonesia yang baik dan benar, kecuali nama file PDF di "sources" yang dipertahankan apa adanya.`

	// List the available PDF sources so the LLM can cite them by exact filename.
	sourceList := "(tidak ada sumber tersedia)"
	if len(chromaSources) > 0 {
		sourceList = "- " + strings.Join(chromaSources, "\n- ")
	}

	user := fmt.Sprintf(`== KONTEKS RISET DARI KNOWLEDGE BASE (SUMBER UTAMA — WAJIB DIGUNAKAN) ==
%s
== AKHIR KONTEKS RISET ==

Daftar file PDF yang tersedia di knowledge base ini:
%s

Gunakan seluruh teori, fakta, dan kerangka pikir dari Konteks Riset di atas sebagai landasan utama analisis Anda.
Pada field "sources" di JSON, cantumkan HANYA nama file PDF yang benar-benar Anda jadikan referensi dari daftar di atas.

---

## Profil Anak yang Perlu Dianalisis:
- **Nama Anak:** %s
- **Usia:** %d tahun
- **Aktivitas Harian yang Disukai:** %s
- **Hal yang Memicu Antusiasme (Positive Triggers):** %s
- **Kekhawatiran Orang Tua:** %s
- **Tujuan & Harapan Orang Tua:** %s

Berdasarkan Konteks Riset di atas dan profil anak ini, analisis dengan empati dan kembalikan HANYA objek JSON dalam Bahasa Indonesia.`,
		ragContext,
		sourceList,
		p.ChildName,
		p.ChildAge,
		activities,
		p.PositiveTriggers,
		p.ParentAnxiety,
		p.ParentGoals,
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
