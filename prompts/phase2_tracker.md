# Prompt: Implement Longitudinal Tracking & PostgreSQL to Bimbi AI

**Context & Objective**
* **Role:** Principal Go Engineer & Software Architect.
* **Task:** Integrate a relational database (PostgreSQL via GORM) into the existing Clean Architecture Go backend to support longitudinal tracking of children's assessments.
* **Goal:** Create a `Child` master entity and an `Assessment` history entity. Update the RAG service to fetch a child's previous assessment, inject it into the Gemini LLM prompt to analyze progress, and save the new result.

**Step 1: Domain & Models (`internal/domain/`)**
Create new files or update existing ones for GORM models and payload structs:
* **Models (`models.go`):**
  * `Child`: fields `ID` (UUID), `ParentID` (string), `Name` (string), `BirthDate` (time.Time), `CreatedAt`, `UpdatedAt`.
  * `Assessment`: fields `ID` (UUID), `ChildID` (UUID, Foreign Key to Child), `AssessmentDate` (time.Time), `InputPayload` (JSONB), `AIResponse` (JSONB), `CreatedAt`.
* **Payloads (`payloads.go`):**
  * `CreateChildRequest`: `parent_id`, `name`, `birth_date`.
  * Update `AssessmentRequest`: change `child_name` and `child_age` to `child_id` (UUID). Keep `parent_id`, `daily_activities`, `parent_anxiety`, `positive_triggers`, `parent_goals`.
  * Update `InsightResponse` to include: `progress_analysis` (string) and `theoretical_basis` (string).
* **Interfaces (`repository.go`):**
  * Add `ChildRepository` interface (Create, GetByID).
  * Add `AssessmentRepository` interface (Create, GetLastAssessmentByChildID).

**Step 2: Repository Layer (`internal/repository/postgres_repo.go`)**
* Implement the `ChildRepository` and `AssessmentRepository` using GORM and PostgreSQL.
* Ensure `GetLastAssessmentByChildID` returns the single most recent assessment record for a specific child (order by created_at desc).

**Step 3: Service Layer (`internal/service/rag_service.go`)**
Update the RAG flow inside the service:
1. Fetch the last assessment for the given `child_id` using the `AssessmentRepository`.
2. Fetch the RAG context from ChromaDB using `daily_activities` and `positive_triggers`.
3. **Dynamic System Prompt:** Construct the prompt for Gemini. 
   * If there IS a past assessment, instruct the LLM: "Here is the parent's past anxiety: [Past Anxiety]. Here is the new anxiety/observation: [Current Anxiety]. Write a 'progress_analysis' acknowledging the child's behavioral shift."
   * If there is NO past assessment, skip the progress comparison but still provide the 'theoretical_basis'.
4. After receiving the JSON response from the LLM, save both the Request and Response into the `assessments` table via the repository.

**Step 4: Handlers & Routes (`internal/handler/`)**
* Create `child_handler.go` with an endpoint `POST /api/children` to create a new child profile.
* Update `insight_handler.go` (endpoint `POST /api/assessments`) to parse the new request containing `child_id`, call the updated service, and return the final JSON.
* Ensure both handlers are wired correctly in `cmd/api/main.go`.

**Step 5: Home Activity Recommendations with Done Status (`GET /api/assessments/:id/home-activities`)**
* **Context:** After an assessment is generated, the frontend wants to display the AI-recommended `home_activities` to the parent. Each activity must show whether it has already been acted on (i.e., the parent submitted feedback for it). Activities that are `done` can be hidden or marked complete in the UI. The parent can still submit free-text activities via `POST /api/assessments/:id/feedback` — those are not restricted to the recommendation list.
* **Domain (`payloads.go`):**
  * Add `HomeActivityItem` struct: `activity_name` (string), `done` (bool).
  * Add `HomeActivitiesResponse` struct: `assessment_id` (string), `activities` ([]HomeActivityItem).
* **Domain (`service.go`):**
  * Add `GetHomeActivities(ctx, assessmentID string) (*HomeActivitiesResponse, error)` to the `RagService` interface.
* **Domain (`repository.go`):**
  * Add `GetCompletedActivityNamesByAssessmentID(ctx, assessmentID string) (map[string]bool, error)` to the `FeedbackRepository` interface. Returns a set of lowercase activity names that have any submitted feedback entry (any submission = done, regardless of status).
* **Repository (`internal/repository/postgres_feedback_repo.go`):**
  * Implement `GetCompletedActivityNamesByAssessmentID` using GORM `Pluck` on `activity_name` column. Normalize keys to lowercase for case-insensitive matching.
* **Service (`internal/service/rag_service.go`):**
  * Implement `GetHomeActivities`:
    1. Fetch the `Assessment` by ID via `assessmentRepo.GetByID`.
    2. Unmarshal `assessment.AIResponse` into `InsightResponse` to extract `home_activities`.
    3. Call `feedbackRepo.GetCompletedActivityNamesByAssessmentID` to get the done-set.
    4. Build `[]HomeActivityItem` — for each activity name, set `done = completedNames[strings.ToLower(name)]`.
    5. Return `HomeActivitiesResponse`.
* **Handler (`internal/handler/insight_handler.go`):**
  * Add `GetHomeActivities(c *gin.Context)` method to `InsightHandler`. Parse `:id` param, call service, return 200 with `HomeActivitiesResponse`. Return 404 if assessment not found.
* **Router (`cmd/api/main.go`):**
  * Register: `protected.GET("/assessments/:id/home-activities", insightHandler.GetHomeActivities)` (before the feedback POST route).
* **LLM Prompt & Insight Generation (no change):**
  * The `GenerateInsights` flow remains unchanged. The `home_activities` and `learning_hacks` fields already exist in `InsightResponse` and are saved in `ai_response` JSONB. The generate-insight consideration for the next assessment still reads from `action_feedbacks` + previous `assessment` tables, as before.

**Output Generation**
Please write the code step-by-step. Start by outputting the `internal/domain/` files and the GORM models. Do not use placeholder comments for core logic.