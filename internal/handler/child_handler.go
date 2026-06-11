package handler

import (
	"fmt"
	"net/http"
	"time"

	"bimbi-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChildHandler struct {
	childRepo domain.ChildRepository
}

func NewChildHandler(childRepo domain.ChildRepository) *ChildHandler {
	return &ChildHandler{
		childRepo: childRepo,
	}
}

func (h *ChildHandler) CreateChild(c *gin.Context) {
	var req domain.CreateChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userIDStr := fmt.Sprintf("%v", userIDVal)

	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_date_format",
			"message": "Birth date must be in YYYY-MM-DD format",
		})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid user token"})
		return
	}

	child := &domain.Child{
		ParentID:  userUUID,
		Name:      req.Name,
		Gender:    req.Gender,
		BirthDate: birthDate,
	}

	if err := h.childRepo.Create(c.Request.Context(), child); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": "Failed to create child profile",
		})
		return
	}

	c.JSON(http.StatusCreated, child)
}

func (h *ChildHandler) GetChildren(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userIDStr := fmt.Sprintf("%v", userIDVal)

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid user token"})
		return
	}

	children, err := h.childRepo.GetByParentID(c.Request.Context(), userUUID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": "Failed to fetch children",
		})
		return
	}

	c.JSON(http.StatusOK, children)
}

func (h *ChildHandler) GetChild(c *gin.Context) {
	childID := c.Param("id")
	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

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

	c.JSON(http.StatusOK, child)
}

func (h *ChildHandler) UpdateChild(c *gin.Context) {
	childID := c.Param("id")
	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

	var req domain.UpdateChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_payload", "message": err.Error()})
		return
	}

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

	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_date_format", "message": "Birth date must be in YYYY-MM-DD format"})
		return
	}

	child.Name = req.Name
	child.Gender = req.Gender
	child.BirthDate = birthDate

	if err := h.childRepo.Update(c.Request.Context(), child); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": "Failed to update child profile"})
		return
	}

	c.JSON(http.StatusOK, child)
}

func (h *ChildHandler) DeleteChild(c *gin.Context) {
	childID := c.Param("id")
	userIDVal, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", userIDVal)

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

	if err := h.childRepo.Delete(c.Request.Context(), childID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": "Failed to delete child profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Child deleted successfully"})
}
