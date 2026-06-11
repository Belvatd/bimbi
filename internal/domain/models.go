package domain

import (
	"time"

	"github.com/google/uuid"
)

type Child struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ParentID  uuid.UUID `gorm:"type:uuid;not null;index" json:"parent_id"` // Matches User.ID
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Gender    string    `gorm:"type:varchar(20);not null;default:'unknown'" json:"gender"`
	BirthDate time.Time `gorm:"type:date;not null" json:"birth_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Assessment struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ChildID        uuid.UUID `gorm:"type:uuid;not null;index" json:"child_id"` // Foreign Key to Child
	Child          Child     `gorm:"foreignKey:ChildID" json:"-"`
	AssessmentDate time.Time `gorm:"type:timestamp;not null;default:current_timestamp" json:"assessment_date"`
	InputPayload   []byte    `gorm:"type:jsonb;not null" json:"input_payload"` // JSONB
	AIResponse     []byte    `gorm:"type:jsonb;not null" json:"ai_response"`   // JSONB
	CreatedAt      time.Time `json:"created_at"`
}

type ActionFeedback struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	AssessmentID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"assessment_id"` // Foreign Key to Assessment
	Assessment       Assessment `gorm:"foreignKey:AssessmentID" json:"-"`
	ActivityName     string     `gorm:"type:text;not null" json:"activity_name"`
	ParentExperience string     `gorm:"type:text;not null" json:"parent_experience"`
	Status           string     `gorm:"type:varchar(50);not null" json:"status"` // e.g., "completed", "struggled"
	CreatedAt        time.Time  `json:"created_at"`
}
