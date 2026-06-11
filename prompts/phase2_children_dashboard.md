# Prompt: Implement Dashboard API for Child Assessment History

**Context & Objective**
* **Role:** Principal Go Engineer.
* **Task:** Build a new GET endpoint in our Clean Architecture Go backend to fetch and format a child's assessment history into a structured dashboard payload.
* **Database Context:** We have an `assessments` table containing `child_id`, `assessment_date`, `input_payload` (JSONB), and `ai_response` (JSONB).

**Step 1: Domain Layer (`internal/domain/`)**
Create structs that are highly optimized for a frontend dashboard consumption.
* Add these to your payloads/responses file:
  * `DashboardResponse`: containing `child_id` (string), `total_assessments` (int), and `timeline` (array of `TimelineItem`).
  * `TimelineItem`: containing `assessment_id` (string), `date` (time.Time or formatted string), `activities_observed` (array of strings, extracted from `input_payload.daily_activities`), `talent_label` (string, extracted from `ai_response.talent_label`), `progress_summary` (string, extracted from `ai_response.progress_analysis` or `empathetic_analysis`), and `full_response` (the complete parsed JSON of `ai_response` for modal details).

**Step 2: Repository Layer (`internal/repository/postgres_repo.go`)**
* Add a new method to the `AssessmentRepository` interface: `GetAssessmentsByChildID(childID string) ([]Assessment, error)`.
* Implement this method using GORM to fetch all records where `child_id = ?`, ordered by `assessment_date` ASC (chronological order).

**Step 3: Service Layer (`internal/service/`)**
* Add a method `GetChildDashboard(childID string) (domain.DashboardResponse, error)` to the relevant service.
* Logic: 
  1. Call the repository to get the list of assessments.
  2. If empty, return an appropriate error or empty timeline.
  3. Loop through the database records. Unmarshal the `InputPayload` and `AIResponse` JSONB fields into their respective Go structs to extract the specific fields needed for the `TimelineItem`.
  4. Construct and return the final `DashboardResponse`.

**Step 4: Handler & Routing (`internal/handler/` & `cmd/api/main.go`)**
* Create a handler method `GetDashboard` in `child_handler.go` or `insight_handler.go`.
* Extract the `id` from the URL parameter (e.g., `GET /api/children/:id/dashboard`).
* Call the service and return the structured JSON with a `200 OK` status.
* Register the new route in your Gin router setup in `main.go`.

**Output Generation**
Please output the code step-by-step for the Domain, Repository, Service, and Handler layers. Ensure the JSON unmarshaling of the DB columns is handled safely.