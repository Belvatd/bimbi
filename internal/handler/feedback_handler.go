package handler

import (
	"net/http"

	"bimbi-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	feedbackService domain.FeedbackService
}

func NewFeedbackHandler(feedbackService domain.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
	}
}

// SubmitFeedback handles POST /api/children/:id/feedback.
// The :id is the child UUID. Backend resolves the latest assessment automatically —
// no need for the frontend to track or pass assessment_id.
// activity_name can be a recommended activity from GET /api/children/:id/home-activities
// (copy as-is) or any free-text activity the parent did independently.
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	childID := c.Param("id")

	var req domain.SubmitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	if err := h.feedbackService.SubmitActivityFeedback(c.Request.Context(), childID, req); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "no assessment found for child — run an assessment first before submitting feedback" {
			status = http.StatusUnprocessableEntity
		}
		c.JSON(status, gin.H{
			"error":   "service_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback saved successfully",
	})
}
