package domain

type StudentPayload struct {
	StudentName      string `json:"student_name"       binding:"required"`
	Age              int    `json:"age"                binding:"required"`
	ChildObservation string `json:"child_observation"  binding:"required"`
	TeacherNotes     string `json:"teacher_notes"      binding:"required"`
	ParentHopes      string `json:"parent_hopes"       binding:"required"`
}
