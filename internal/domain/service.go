package domain

import "context"

type RagService interface {
	GenerateInsights(ctx context.Context, payload AssessmentRequest) (*InsightResponse, error)
}

type AuthService interface {
	Register(email, password string) (*User, error)
	Login(email, password string) (string, error) // Returns JWT token
}
