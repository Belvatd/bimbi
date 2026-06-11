package service

import (
	"context"
	"fmt"

	"bimbi-backend/internal/domain"
)

type feedbackService struct {
	feedbackRepo   domain.FeedbackRepository
	assessmentRepo domain.AssessmentRepository
}

func NewFeedbackService(feedbackRepo domain.FeedbackRepository, assessmentRepo domain.AssessmentRepository) domain.FeedbackService {
	return &feedbackService{
		feedbackRepo:   feedbackRepo,
		assessmentRepo: assessmentRepo,
	}
}

// SubmitActivityFeedback resolves the latest assessment for the given child, then saves
// the feedback entry. This keeps the frontend decoupled from assessment_id — it only
// needs to know child_id. Free-text activities are supported: activity_name is stored
// as-is and matched case-insensitively against home_activities in the done-status check.
func (s *feedbackService) SubmitActivityFeedback(ctx context.Context, childID string, req domain.SubmitFeedbackRequest) error {
	// 1. Resolve latest assessment for this child
	assessment, err := s.assessmentRepo.GetLastAssessmentByChildID(ctx, childID)
	if err != nil {
		return fmt.Errorf("failed to fetch latest assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("no assessment found for child — run an assessment first before submitting feedback")
	}

	// 2. Save feedback linked to the latest assessment
	feedback := &domain.ActionFeedback{
		AssessmentID:     assessment.ID,
		ActivityName:     req.ActivityName,
		ParentExperience: req.ParentExperience,
		Status:           req.Status,
	}

	return s.feedbackRepo.CreateFeedback(ctx, feedback)
}
