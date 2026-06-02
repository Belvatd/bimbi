package domain

// InsightResponse is the B2C parent-focused output payload.
type InsightResponse struct {
	TalentLabel        string   `json:"talent_label"`
	EmpatheticAnalysis string   `json:"empathetic_analysis"`
	TheoreticalBasis   string   `json:"theoretical_basis"` // psychological/educational theory grounding the analysis
	HomeActivities     []string `json:"home_activities"`
	LearningHacks      []string `json:"learning_hacks"`
	Sources            []string `json:"sources"` // RAG source PDF filenames
}
