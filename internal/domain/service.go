package domain

import "context"

type RagService interface {
	GenerateInsights(ctx context.Context, payload AssessmentRequest) (*InsightResponse, error)
	GetChildDashboard(ctx context.Context, childID string) (DashboardResponse, error)
	GetHomeActivities(ctx context.Context, childID string) (*HomeActivitiesResponse, error)
}

type AuthService interface {
	Register(email, password string) (*User, error)
	Login(email, password string) (string, error) // Returns JWT token
}

type FeedbackService interface {
	// SubmitActivityFeedback saves feedback for an activity under the child's latest assessment.
	// childID is used to resolve the latest assessment automatically — frontend doesn't need to track assessment_id.
	// activity_name can be a recommended activity from home-activities OR a free-text entry.
	SubmitActivityFeedback(ctx context.Context, childID string, req SubmitFeedbackRequest) error
}
