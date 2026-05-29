package repository

import (
	"context"
	"fmt"

	"bimbi-backend/internal/domain"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

type llmRepo struct {
	client *googleai.GoogleAI
}

func NewLLMRepo(ctx context.Context, apiKey string) (domain.LLMRepo, error) {
	client, err := googleai.New(ctx,
		googleai.WithAPIKey(apiKey),
		googleai.WithDefaultModel("gemini-2.5-flash"),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Gemini: %w", err)
	}

	return &llmRepo{client: client}, nil
}

func (r *llmRepo) Call(ctx context.Context, prompt string) (string, error) {
	response, err := r.client.Call(ctx, prompt,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(8192),
	)
	if err != nil {
		return "", fmt.Errorf("llm call: %w", err)
	}
	return response, nil
}
