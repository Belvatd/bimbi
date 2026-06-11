package handler

import (
	"fmt"
	"net/http"

	"bimbi-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AssessmentHandler struct {
	assessmentRepo domain.AssessmentRepository
	childRepo      domain.ChildRepository
}

func NewAssessmentHandler(assessmentRepo domain.AssessmentRepository, childRepo domain.ChildRepository) *AssessmentHandler {
	return &AssessmentHandler{
		assessmentRepo: assessmentRepo,
		childRepo:      childRepo,
	}
}

func (h *AssessmentHandler) GetAssessments(c *gin.Context) {
	childID := c.Query("child_id")
	if childID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "child_id query parameter is required"})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

	// Verify child belongs to user
	child, err := h.childRepo.GetByID(c.Request.Context(), childID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Child not found"})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil || child.ParentID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Access denied"})
		return
	}

	assessments, err := h.assessmentRepo.GetByChildID(c.Request.Context(), childID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": "Failed to fetch assessments"})
		return
	}

	c.JSON(http.StatusOK, assessments)
}

func (h *AssessmentHandler) GetAssessment(c *gin.Context) {
	assessmentID := c.Param("id")
	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

	assessment, err := h.assessmentRepo.GetByID(c.Request.Context(), assessmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Assessment not found"})
		return
	}

	// Verify child belongs to user
	child, err := h.childRepo.GetByID(c.Request.Context(), assessment.ChildID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Child associated with assessment not found"})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil || child.ParentID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, assessment)
}

func (h *AssessmentHandler) DeleteAssessment(c *gin.Context) {
	assessmentID := c.Param("id")
	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

	assessment, err := h.assessmentRepo.GetByID(c.Request.Context(), assessmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Assessment not found"})
		return
	}

	// Verify child belongs to user
	child, err := h.childRepo.GetByID(c.Request.Context(), assessment.ChildID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "Child associated with assessment not found"})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil || child.ParentID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Access denied"})
		return
	}

	if err := h.assessmentRepo.Delete(c.Request.Context(), assessmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": "Failed to delete assessment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assessment deleted successfully"})
}
