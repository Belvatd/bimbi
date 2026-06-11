# Prompt: Implement Actionable Item Feedback Loop in Bimbi AI

**Context & Objective**
* **Role:** Principal Go Engineer.
* **Task:** Implement a feedback loop for actionable items (`home_activities`). Parents can submit their experience after trying an activity. The activity can be chosen from the list of recommended activities (which are refreshed every time a new assessment is generated) OR they can input a free-text activity they did on their own. This feedback is SAVED statically and ONLY used as context for the NEXT assessment's LLM prompt. It does NOT trigger a new LLM response upon submission.
* **Goal:** Create a new `ActionFeedback` entity/table. Create a simple POST endpoint to save this feedback. Update the existing RAG Assessment Service to fetch previous feedbacks and inject them into the Gemini System Prompt for the next assessment.

**Step 1: Domain Models & Payloads (`internal/domain/`)**
* **Model (`models.go`):** Add `ActionFeedback` struct for GORM.
  * Fields: `ID` (UUID), `AssessmentID` (UUID, Foreign Key), `ActivityName` (string, the exact actionable item text), `ParentExperience` (string), `Status` (string, e.g., "completed", "struggled"), `CreatedAt`.
* **Payload (`payloads.go`):** Add `SubmitFeedbackRequest`.
  * Fields: `activity_name` (can be a recommendation from the assessment OR free-text from the user), `parent_experience`, `status`.
* **Interface (`repository.go`):** Add `FeedbackRepository` interface with methods: `CreateFeedback(feedback ActionFeedback) error` and `GetFeedbacksByAssessmentID(assessmentID string) ([]ActionFeedback, error)`.

**Step 2: Repository Layer (`internal/repository/postgres_repo.go`)**
* Implement the `FeedbackRepository` methods using GORM.

**Step 3: Service Layer - Saving Feedback (`internal/service/feedback_service.go`)**
* Create a simple service method `SubmitActivityFeedback(assessmentID string, req SubmitFeedbackRequest) error`.
* It only maps the request to the `ActionFeedback` entity and calls the repository to save it. Return success. Do NOT call the LLM here.

**Step 4: Service Layer - Updating RAG Context (`internal/service/rag_service.go`)**
* Modify the existing `GenerateAssessment` method (the one that calls Gemini).
* **New Logic Flow:**
  1. Fetch the last assessment for the child.
  2. If a last assessment exists, fetch all `ActionFeedback` records associated with that `last_assessment.ID`.
  3. **Prompt Injection:** If feedbacks exist, construct a "Memory String". Example: "Last month, the parent tried these activities: 1. [ActivityName] - Parent noted: [ParentExperience]. Status: [Status]."
  4. Inject this Memory String into the Gemini System Prompt with instructions: "Use the parent's past experiences to evaluate progress and ensure the NEW 'home_activities' and 'learning_hacks' are adjusted. If they struggled, provide easier alternatives. If they succeeded, provide the next level."
  5. Fetch ChromaDB context and call LLM as usual.

**Step 5: Handlers & Routes (`internal/handler/`)**
* Create `feedback_handler.go` with endpoint `POST /api/assessments/:assessment_id/feedback`.
* Bind JSON, call `SubmitActivityFeedback` service, and return a simple `200 OK` with a success message (e.g., {"message": "Feedback saved successfully"}).
* Wire the new handler and repository in `cmd/api/main.go`.

**Output Generation**
Output the code step-by-step. Focus heavily on Step 4 (the `rag_service.go` update), ensuring the string builder effectively integrates the past feedbacks into the prompt before making the LangChain/Gemini API call.