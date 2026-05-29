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
