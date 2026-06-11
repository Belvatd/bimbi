package domain

import (
	"context"
)

type VectorRepo interface {
	Query(ctx context.Context, queryText string, topK int) (ragContext string, sources []string, err error)
}

type LLMRepo interface {
	Call(ctx context.Context, prompt string) (response string, err error)
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
}

type ChildRepository interface {
	Create(ctx context.Context, child *Child) error
	GetByID(ctx context.Context, id string) (*Child, error)
	GetByParentID(ctx context.Context, parentID string) ([]*Child, error)
	Update(ctx context.Context, child *Child) error
	Delete(ctx context.Context, id string) error
}

type AssessmentRepository interface {
	Create(ctx context.Context, assessment *Assessment) error
	GetLastAssessmentByChildID(ctx context.Context, childID string) (*Assessment, error)
	GetByChildID(ctx context.Context, childID string) ([]*Assessment, error)
	GetAssessmentsByChildID(ctx context.Context, childID string) ([]*Assessment, error)
	GetByID(ctx context.Context, id string) (*Assessment, error)
	Delete(ctx context.Context, id string) error
}

type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *ActionFeedback) error
	GetFeedbacksByAssessmentID(ctx context.Context, assessmentID string) ([]*ActionFeedback, error)
	// GetCompletedActivityNamesByAssessmentID returns a set of activity_names that have been
	// submitted as feedback for the given assessment. Any submitted activity is considered "done".
	GetCompletedActivityNamesByAssessmentID(ctx context.Context, assessmentID string) (map[string]bool, error)
}
