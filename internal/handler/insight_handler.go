package handler

import (
	"fmt"
	"net/http"

	"bimbi-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type InsightHandler struct {
	ragService domain.RagService
}

func NewInsightHandler(ragService domain.RagService) *InsightHandler {
	return &InsightHandler{
		ragService: ragService,
	}
}

func (h *InsightHandler) GenerateInsights(c *gin.Context) {
	var payload domain.AssessmentRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	ctx := c.Request.Context()

	insight, err := h.ragService.GenerateInsights(ctx, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "service_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, insight)
}

func (h *InsightHandler) GetDashboard(c *gin.Context) {
	childID := c.Param("id")

	ctx := c.Request.Context()
	dashboard, err := h.ragService.GetChildDashboard(ctx, childID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "service_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetHomeActivities handles GET /api/children/:id/home-activities.
// It fetches the LATEST assessment for the child and returns its recommended home_activities,
// each annotated with a `done` flag (true = parent already submitted feedback for it).
// The response also includes `assessment_id` so the frontend knows which assessment to
// post feedback to via POST /api/assessments/:assessment_id/feedback.
// Free-text activities can always be submitted regardless of this list.
func (h *InsightHandler) GetHomeActivities(c *gin.Context) {
	childID := c.Param("id")

	ctx := c.Request.Context()
	resp, err := h.ragService.GetHomeActivities(ctx, childID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "service_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
