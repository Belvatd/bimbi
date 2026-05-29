package domain

type InsightResponse struct {
	TalentLabel            string   `json:"talent_label"`
	PersonalityAnalysis    string   `json:"personality_analysis"`
	ParentRecommendations  []string `json:"parent_recommendations"`
	TeacherRecommendations []string `json:"teacher_recommendations"`
	Sources                []string `json:"sources"` // nama file PDF sumber konteks RAG
}
