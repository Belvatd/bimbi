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
	var payload domain.StudentPayload
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
