package domain

import "time"

type CreateChildRequest struct {
	Name      string `json:"name" binding:"required"`
	Gender    string `json:"gender" binding:"required"`
	BirthDate string `json:"birth_date" binding:"required"` // Format: YYYY-MM-DD
}

type UpdateChildRequest struct {
	Name      string `json:"name" binding:"required"`
	Gender    string `json:"gender" binding:"required"`
	BirthDate string `json:"birth_date" binding:"required"` // Format: YYYY-MM-DD
}

// AssessmentRequest is the B2C parent-focused input payload.
type AssessmentRequest struct {
	ChildID          string   `json:"child_id" binding:"required,uuid"`
	DailyActivities  []string `json:"daily_activities" binding:"required,min=1"`
	ParentAnxiety    string   `json:"parent_anxiety" binding:"required"`
	PositiveTriggers string   `json:"positive_triggers" binding:"required"`
	ParentGoals      string   `json:"parent_goals" binding:"required"`
}

// InsightResponse is the B2C parent-focused output payload.
type InsightResponse struct {
	TalentLabel        string   `json:"talent_label"`
	EmpatheticAnalysis string   `json:"empathetic_analysis"`
	TheoreticalBasis   string   `json:"theoretical_basis"` // psychological/educational theory grounding the analysis
	ProgressAnalysis   string   `json:"progress_analysis,omitempty"` // acknowledges the child's behavioral shift compared to previous assessment
	HomeActivities     []string `json:"home_activities"`
	LearningHacks      []string `json:"learning_hacks"`
	Sources            []string `json:"sources"` // RAG source PDF filenames
}

type TimelineItem struct {
	AssessmentID       string          `json:"assessment_id"`
	Date               time.Time       `json:"date"`
	ActivitiesObserved []string        `json:"activities_observed"`
	TalentLabel        string          `json:"talent_label"`
	ProgressSummary    string          `json:"progress_summary"`
	FullResponse       InsightResponse `json:"full_response"`
}

type DashboardResponse struct {
	ChildID          string         `json:"child_id"`
	TotalAssessments int            `json:"total_assessments"`
	Timeline         []TimelineItem `json:"timeline"`
}

type SubmitFeedbackRequest struct {
	ActivityName     string `json:"activity_name" binding:"required"`
	ParentExperience string `json:"parent_experience" binding:"required"`
	Status           string `json:"status" binding:"required,oneof=completed struggled"`
}

// HomeActivityItem represents a single recommended home activity with its completion status.
type HomeActivityItem struct {
	ActivityName string `json:"activity_name"`
	Done         bool   `json:"done"`
}

// HomeActivitiesResponse is the response payload for GET /api/assessments/:id/home-activities.
type HomeActivitiesResponse struct {
	AssessmentID string             `json:"assessment_id"`
	Activities   []HomeActivityItem `json:"activities"`
}
