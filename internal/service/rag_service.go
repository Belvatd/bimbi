package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"bimbi-backend/internal/domain"

	"github.com/google/uuid"
)

// trailingCommaRe matches trailing commas before a closing bracket/brace,
// which LLMs frequently emit but are invalid in strict JSON.
var trailingCommaRe = regexp.MustCompile(`,\s*([}\]])`)

type ragService struct {
	vectorRepo     domain.VectorRepo
	llmRepo        domain.LLMRepo
	childRepo      domain.ChildRepository
	assessmentRepo domain.AssessmentRepository
	feedbackRepo   domain.FeedbackRepository
}

func NewRagService(vectorRepo domain.VectorRepo, llmRepo domain.LLMRepo, childRepo domain.ChildRepository, assessmentRepo domain.AssessmentRepository, feedbackRepo domain.FeedbackRepository) domain.RagService {
	return &ragService{
		vectorRepo:     vectorRepo,
		llmRepo:        llmRepo,
		childRepo:      childRepo,
		assessmentRepo: assessmentRepo,
		feedbackRepo:   feedbackRepo,
	}
}

func (s *ragService) GenerateInsights(ctx context.Context, payload domain.AssessmentRequest) (*domain.InsightResponse, error) {
	// 1. Fetch child profile
	child, err := s.childRepo.GetByID(ctx, payload.ChildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch child profile: %w", err)
	}

	// Calculate age (simplified)
	age := time.Now().Year() - child.BirthDate.Year()
	if time.Now().YearDay() < child.BirthDate.YearDay() {
		age--
	}
	if age < 0 {
		age = 0
	}

	// 2. Fetch last assessment
	lastAssessment, err := s.assessmentRepo.GetLastAssessmentByChildID(ctx, payload.ChildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch last assessment: %w", err)
	}

	// 3. Fetch RAG Context
	queryText := s.buildQueryText(payload)
	ragContext, chromaSources, err := s.vectorRepo.Query(ctx, queryText, 10)
	if err != nil {
		ragContext = "Tidak ada konteks knowledge base eksternal. Gunakan pengetahuan ahli Anda."
		chromaSources = []string{}
	}

	// 4. Build prompt
	var pastAnxiety string
	var pastFeedbacks []*domain.ActionFeedback
	if lastAssessment != nil {
		// Attempt to parse past anxiety from input payload
		var pastPayload domain.AssessmentRequest
		if err := json.Unmarshal(lastAssessment.InputPayload, &pastPayload); err == nil {
			pastAnxiety = pastPayload.ParentAnxiety
		}
		
		if s.feedbackRepo != nil {
			pastFeedbacks, _ = s.feedbackRepo.GetFeedbacksByAssessmentID(ctx, lastAssessment.ID.String())
		}
	}

	fullPrompt := s.buildPrompt(ragContext, chromaSources, payload, child.Name, age, pastAnxiety, pastFeedbacks)

	// 5. Call LLM
	rawResponse, err := s.llmRepo.Call(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm error: %w", err)
	}

	// 6. Parse response
	insight, err := s.parseInsightJSON(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// 7. Save Assessment to DB
	inputJSON, _ := json.Marshal(payload)
	aiJSON, _ := json.Marshal(insight)

	parsedChildID, _ := uuid.Parse(payload.ChildID)
	assessment := &domain.Assessment{
		ChildID:        parsedChildID,
		AssessmentDate: time.Now(),
		InputPayload:   inputJSON,
		AIResponse:     aiJSON,
	}
	if err := s.assessmentRepo.Create(ctx, assessment); err != nil {
		// Just log error, don't fail the request since insight is already generated
		fmt.Printf("warning: failed to save assessment to db: %v\n", err)
	}

	return insight, nil
}

func (s *ragService) GetChildDashboard(ctx context.Context, childID string) (domain.DashboardResponse, error) {
	assessments, err := s.assessmentRepo.GetAssessmentsByChildID(ctx, childID)
	if err != nil {
		return domain.DashboardResponse{}, fmt.Errorf("failed to fetch assessments: %w", err)
	}

	var timeline []domain.TimelineItem
	for _, a := range assessments {
		var input domain.AssessmentRequest
		if err := json.Unmarshal(a.InputPayload, &input); err != nil {
			fmt.Printf("warning: failed to unmarshal input payload for assessment %s: %v\n", a.ID, err)
			continue
		}

		var ai domain.InsightResponse
		if err := json.Unmarshal(a.AIResponse, &ai); err != nil {
			fmt.Printf("warning: failed to unmarshal ai response for assessment %s: %v\n", a.ID, err)
			continue
		}

		progressSummary := ai.ProgressAnalysis
		if progressSummary == "" {
			progressSummary = ai.EmpatheticAnalysis
		}

		timeline = append(timeline, domain.TimelineItem{
			AssessmentID:       a.ID.String(),
			Date:               a.AssessmentDate,
			ActivitiesObserved: input.DailyActivities,
			TalentLabel:        ai.TalentLabel,
			ProgressSummary:    progressSummary,
			FullResponse:       ai,
		})
	}

	if timeline == nil {
		timeline = []domain.TimelineItem{}
	}

	return domain.DashboardResponse{
		ChildID:          childID,
		TotalAssessments: len(timeline),
		Timeline:         timeline,
	}, nil
}

// GetHomeActivities returns the recommended home_activities from the LATEST assessment
// for the given child, each annotated with a `done` flag based on whether the parent
// has already submitted feedback for that activity via POST /api/assessments/:id/feedback.
// The response includes assessment_id so the frontend knows which assessment to post feedback to.
func (s *ragService) GetHomeActivities(ctx context.Context, childID string) (*domain.HomeActivitiesResponse, error) {
	// 1. Fetch the latest assessment for this child
	assessment, err := s.assessmentRepo.GetLastAssessmentByChildID(ctx, childID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest assessment: %w", err)
	}
	if assessment == nil {
		return &domain.HomeActivitiesResponse{
			AssessmentID: "",
			Activities:   []domain.HomeActivityItem{},
		}, nil
	}

	// 2. Parse the AI response to extract home_activities
	var aiResp domain.InsightResponse
	if err := json.Unmarshal(assessment.AIResponse, &aiResp); err != nil {
		return nil, fmt.Errorf("failed to parse assessment AI response: %w", err)
	}

	assessmentID := assessment.ID.String()

	// 3. Fetch completed activity names for this assessment
	completedNames, err := s.feedbackRepo.GetCompletedActivityNamesByAssessmentID(ctx, assessmentID)
	if err != nil {
		// Non-fatal: return activities without done status rather than failing entirely
		fmt.Printf("warning: failed to fetch completed activity names: %v\n", err)
		completedNames = map[string]bool{}
	}

	// 4. Build response — mark each activity done if it has a feedback entry
	items := make([]domain.HomeActivityItem, 0, len(aiResp.HomeActivities))
	for _, name := range aiResp.HomeActivities {
		done := completedNames[strings.ToLower(strings.TrimSpace(name))]
		items = append(items, domain.HomeActivityItem{
			ActivityName: name,
			Done:         done,
		})
	}

	return &domain.HomeActivitiesResponse{
		AssessmentID: assessmentID,
		Activities:   items,
	}, nil
}

func (s *ragService) buildQueryText(p domain.AssessmentRequest) string {
	activities := strings.Join(p.DailyActivities, ", ")
	return fmt.Sprintf(
		"Child likes: %s. Engaged when: %s.",
		activities,
		p.PositiveTriggers,
	)
}

func (s *ragService) buildPrompt(ragContext string, chromaSources []string, p domain.AssessmentRequest, childName string, childAge int, pastAnxiety string, pastFeedbacks []*domain.ActionFeedback) string {
	activities := strings.Join(p.DailyActivities, ", ")

	system := `Anda adalah psikolog anak dan pakar parenting modern yang sangat empatik.
Tugas Anda adalah membantu orang tua memahami potensi tersembunyi anak mereka berdasarkan observasi sehari-hari di rumah.

Nada bicara: Hangat, menenangkan, dan praktis. Hindari jargon akademis.

== ATURAN PENGGUNAAN KONTEKS (WAJIB DIPATUHI) ==
Anda akan menerima "Konteks Riset" yang diambil dari knowledge base internal (ChromaDB).
Setiap blok konteks diberi label "[Konteks N — Sumber: nama_file.pdf]" yang menunjukkan dari PDF mana teks tersebut berasal.
Anda HARUS menggunakan konteks tersebut sebagai sumber pengetahuan UTAMA dan PERTAMA.
Semua analisis talent_label, empathetic_analysis, theoretical_basis, home_activities, dan learning_hacks HARUS berakar dari teori dan fakta yang ada di dalam Konteks Riset tersebut.
Jika Konteks Riset memuat teori Multiple Intelligences atau gaya belajar tertentu, gunakan terminologi dan kerangka pikir dari sana.
Hanya gunakan pengetahuan umum Anda sebagai PELENGKAP jika konteks tidak mencakup aspek tertentu. Jangan pernah mengabaikan konteks yang diberikan.
Untuk field "sources": cantumkan HANYA nama file PDF (dari label "Sumber:") yang BENAR-BENAR Anda gunakan sebagai referensi dalam analisis ini. Jangan cantumkan PDF yang tidak Anda rujuk.

🇮🇩 WAJIB: SELURUH respons HARUS dalam Bahasa Indonesia. Tidak ada kata dalam bahasa lain.
Pastikan jawaban padat, jelas, dan ringkas agar tidak terpotong.

CRITICAL: Balas HANYA dengan objek JSON yang valid. Tidak ada markdown, tidak ada code fence, tidak ada penjelasan di luar JSON.
Struktur JSON yang diperlukan:
{
  "talent_label": "<label kecerdasan utama anak dalam Bahasa Indonesia — harus berdasarkan teori di Konteks Riset>",
  "empathetic_analysis": "<Dua paragraf dalam Bahasa Indonesia yang BERAKAR dari Konteks Riset. Paragraf pertama: validasi kekhawatiran orang tua dengan empati. Paragraf kedua: bingkai ulang kekhawatiran tersebut sebagai potensi bakat tersembunyi.>",
  "theoretical_basis": "<Penjelasan singkat landasan teori psikologi atau pendidikan yang mendasari analisis ini — berdasarkan Konteks Riset dan profil anak.>",`
	
	if pastAnxiety != "" {
		system += "\n  \"progress_analysis\": \"<Satu paragraf yang mengakui perubahan perilaku anak. Bandingkan kecemasan masa lalu (Past Anxiety) dengan kecemasan saat ini (Current Anxiety). Berikan dorongan semangat kepada orang tua.>\",\n"
	}

	system += `  "home_activities": [
    "<Rekomendasi aktivitas rumah 1>",
    "<Rekomendasi aktivitas rumah 2>",
    "<Rekomendasi aktivitas rumah 3>"
  ],
  "learning_hacks": [
    "<Tips mendampingi gaya belajar 1>",
    "<Tips mendampingi gaya belajar 2>"
  ],
  "sources": ["<nama_file.pdf yang benar-benar Anda jadikan referensi>"]
}

INGAT: Semua nilai string HARUS dalam Bahasa Indonesia yang baik dan benar, kecuali nama file PDF di "sources" yang dipertahankan apa adanya.`

	if len(pastFeedbacks) > 0 {
		var feedbackStrings []string
		for i, fb := range pastFeedbacks {
			feedbackStrings = append(feedbackStrings, fmt.Sprintf("%d. [%s] - Parent noted: [%s]. Status: [%s].", i+1, fb.ActivityName, fb.ParentExperience, fb.Status))
		}
		feedbackSummary := strings.Join(feedbackStrings, "\n")
		
		system += fmt.Sprintf("\n\n== FEEDBACK AKTIVITAS SEBELUMNYA ==\nPada assessment sebelumnya, orang tua mencoba aktivitas ini:\n%s\n\nInstruksi Tambahan: Gunakan pengalaman masa lalu orang tua ini untuk mengevaluasi kemajuan dan memastikan 'home_activities' dan 'learning_hacks' YANG BARU disesuaikan. Jika mereka kesulitan (struggled), berikan alternatif yang lebih mudah. Jika mereka berhasil (completed), berikan tingkat berikutnya.", feedbackSummary)
	}

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
- **Hal yang Memicu Antusiasme (Positive Triggers):** %s`,
		ragContext,
		sourceList,
		childName,
		childAge,
		activities,
		p.PositiveTriggers,
	)

	if pastAnxiety != "" {
		user += fmt.Sprintf("\n- **Kekhawatiran Masa Lalu (Assessment Sebelumnya):** %s", pastAnxiety)
	}

	user += fmt.Sprintf(`
- **Kekhawatiran Saat Ini (Current Anxiety):** %s
- **Tujuan & Harapan Orang Tua:** %s

Berdasarkan Konteks Riset di atas dan profil anak ini, analisis dengan empati dan kembalikan HANYA objek JSON dalam Bahasa Indonesia.`,
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
	if start == -1 {
		return nil, fmt.Errorf("no valid JSON object found in LLM response")
	}

	depth, end := 0, -1
	inStr, esc := false, false
	for i := start; i < len(cleaned); i++ {
		ch := cleaned[i]
		if esc {
			esc = false
			continue
		}
		if ch == '\\' && inStr {
			esc = true
			continue
		}
		if ch == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end != -1 {
			break
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("no valid JSON object found in LLM response")
	}
	jsonStr := cleaned[start : end+1]

	jsonStr = trailingCommaRe.ReplaceAllString(jsonStr, "$1")

	var insight domain.InsightResponse
	if err := json.Unmarshal([]byte(jsonStr), &insight); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	return &insight, nil
}
