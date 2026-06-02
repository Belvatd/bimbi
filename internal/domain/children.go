package domain

// AssessmentRequest is the B2C parent-focused input payload.
type AssessmentRequest struct {
	ChildName        string   `json:"child_name"        binding:"required"`
	ChildAge         int      `json:"child_age"         binding:"required"`
	DailyActivities  []string `json:"daily_activities"  binding:"required,min=1"`
	ParentAnxiety    string   `json:"parent_anxiety"    binding:"required"`
	PositiveTriggers string   `json:"positive_triggers" binding:"required"`
	ParentGoals      string   `json:"parent_goals"      binding:"required"`
}
