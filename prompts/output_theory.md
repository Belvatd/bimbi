# Prompt: Add Theoretical Basis Output to Bimbi AI

**Context & Objective**
* **Role:** Principal Go Engineer & Prompt Engineer.
* **Task:** Extend the current B2C JSON response structure in the Go backend to include a new field that explains the psychological or educational theory grounding the AI's analysis.
* **Goal:** Update the Domain struct and the LLM System Prompt in the Service layer to ensure Gemini returns this new field accurately based on the RAG context.

**Step 1: Update Domain Struct (`internal/domain/`)**
* Locate the Response Struct (e.g., `InsightResponse`) used for the LLM output.
* Add a new field exactly like this:
  * `theoretical_basis` (string): A brief explanation of the psychological, developmental, or educational theory that supports the analysis (e.g., "Berdasarkan teori Kecerdasan Majemuk Howard Gardner...", or "Mengacu pada pendekatan Montessori...").

**Step 2: Update the Service & Prompt Engineering (`internal/service/rag_service.go`)**
* Find the system prompt string that instructs the Gemini LLM.
* Add a new specific instruction for the LLM regarding the `theoretical_basis` field.
* **New Prompt Instruction:** "You must include a 'theoretical_basis' field in your JSON output. Based on the provided RAG Context and the child's profile, identify and explain the specific psychological or educational theory (such as Multiple Intelligences, Cognitive Development theories, etc.) that validates your 'empathetic_analysis' and recommendations. Start the sentence with 'Berdasarkan teori...' or 'Mengacu pada...'. Keep it easy for parents to understand, but scientifically accurate."

**Output Generation**
Please output the updated code ONLY for the files that require changes:
1. The specific Go file in `internal/domain/` containing the response struct.
2. `internal/service/rag_service.go` (specifically showing the updated struct mapping and the modified system prompt).

Ensure the existing dependency injection and overall structure remain intact.