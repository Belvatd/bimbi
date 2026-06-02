# Prompt: Update Go Backend to B2C Parent-Focused Payload

**Context & Objective**
* **Role:** Principal Go Engineer & Prompt Engineer.
* **Task:** Update the existing modular Go codebase for 'Bimbi AI' to reflect a new B2C (Parent-focused) JSON payload. The old school-focused payload (teacher_notes, etc.) is deprecated.
* **Goal:** Modify the Domain structs, the Handler validation, the RAG search query, and the LLM System Prompt to cater exclusively to parents observing their children at home.

**Step 1: Update Domain Structs (`internal/domain/`)**
Please update the request and response structs to match this exact schema.

* **Request Struct (e.g., `ChildProfile` or `AssessmentRequest`):**
    * `child_name` (string)
    * `child_age` (int)
    * `daily_activities` (array of strings)
    * `parent_anxiety` (string)
    * `positive_triggers` (string)
    * `parent_goals` (string)

* **Response Struct (e.g., `InsightResponse`):**
    * `talent_label` (string)
    * `empathetic_analysis` (string, 2 paragraphs. Must validate the parent's anxiety first, then reframe it as a potential hidden talent based on the activities and positive triggers).
    * `home_activities` (array of 3 highly actionable string recommendations that use common household items/situations).
    * `learning_hacks` (array of 2 string recommendations on how to handle the child's learning style at home).

**Step 2: Update the Handler (`internal/handler/insight_handler.go`)**
* Refactor the JSON binding and validation to accept the new Request Struct.
* Ensure it passes the new struct to the Service layer correctly.

**Step 3: Update the Service & Prompt Engineering (`internal/service/rag_service.go`)**
* **Vector Search Query:** When querying ChromaDB, construct the search string by combining `daily_activities` and `positive_triggers`. (e.g., "Child likes: [activities]. Engaged when: [triggers]").
* **System Prompt Rewrite:** Completely rewrite the Gemini system prompt. 
    * *Persona:* Act as a highly empathetic, modern child psychologist and parenting expert.
    * *Tone:* Warm, reassuring, practical. No academic jargon.
    * *Instruction:* Read the parent's `parent_anxiety` and use the provided RAG Context to reassure them. Explain how the `daily_activities` and `positive_triggers` indicate a specific Multiple Intelligence or learning style. Output strictly in the new Response Struct JSON format.

**Output Generation**
Please output the fully updated code for:
1.  `internal/domain/` (The updated structs)
2.  `internal/handler/insight_handler.go`
3.  `internal/service/rag_service.go`

Do not break the existing dependency injection setup.