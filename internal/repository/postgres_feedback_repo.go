package repository

import (
	"context"
	"strings"

	"bimbi-backend/internal/domain"

	"gorm.io/gorm"
)

type postgresFeedbackRepo struct {
	db *gorm.DB
}

func NewPostgresFeedbackRepo(db *gorm.DB) domain.FeedbackRepository {
	return &postgresFeedbackRepo{db: db}
}

func (r *postgresFeedbackRepo) CreateFeedback(ctx context.Context, feedback *domain.ActionFeedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

func (r *postgresFeedbackRepo) GetFeedbacksByAssessmentID(ctx context.Context, assessmentID string) ([]*domain.ActionFeedback, error) {
	var feedbacks []*domain.ActionFeedback
	if err := r.db.WithContext(ctx).Where("assessment_id = ?", assessmentID).Order("created_at asc").Find(&feedbacks).Error; err != nil {
		return nil, err
	}
	return feedbacks, nil
}

func (r *postgresFeedbackRepo) GetCompletedActivityNamesByAssessmentID(ctx context.Context, assessmentID string) (map[string]bool, error) {
	var names []string
	if err := r.db.WithContext(ctx).
		Model(&domain.ActionFeedback{}).
		Where("assessment_id = ?", assessmentID).
		Pluck("activity_name", &names).Error; err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(names))
	for _, n := range names {
		result[strings.ToLower(strings.TrimSpace(n))] = true
	}
	return result, nil
}
