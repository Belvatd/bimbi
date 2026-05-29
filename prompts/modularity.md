# Prompt: Go Clean Architecture Refactoring for Bimbi AI

**Context & Objective**
* **Role:** You are a Principal Go Engineer and Software Architect.
* **Task:** Refactor an existing single-file MVP Go backend into a highly scalable, modular Clean Architecture following the `golang-standards/project-layout`.
* **Application:** 'Bimbi AI' - an EdTech backend handling text-based student observation data to generate psychological insights using a RAG pipeline (No K-Means, pure RAG).
* **Tech Stack:** Go 1.21+, `gin-gonic/gin`, `tmc/langchaingo`, Google Gemini 1.5 Flash, ChromaDB (Local Docker), `joho/godotenv`.

**Core Architectural Rules to Enforce**
* **Strict Layer Isolation:** The `domain` layer must have zero external dependencies (no Gin, no LangChain). It only contains Go structs and interface definitions.
* **Dependency Injection (DI):** Do not instantiate databases or LLM clients inside handlers or services. Pass them via constructor functions (e.g., `NewRagService(repo ...)`).
* **Interface-Driven Design:** The `service` layer must rely on interfaces to communicate with the `repository` layer, not concrete implementations.
* **No Global State:** Absolutely no global variables for database connections, routers, or LLM clients. 
* **Single Responsibility:** Each file must do exactly one thing (e.g., HTTP routing is strictly in `handler`, prompting logic is strictly in `service`).

**Target Directory Structure**
Please generate the code following this exact tree:

```text
/
├── cmd/
│   ├── api/
│   │   └── main.go                  # API Entry point. Loads config, wires dependencies, starts Gin server.
│   └── ingester/
│       └── main.go                  # CLI Entry point for running the PDF ingestion process.
├── internal/
│   ├── config/
│   │   └── config.go                # Defines Config struct and loads .env variables.
│   ├── domain/
│   │   ├── student.go               # Payload struct: StudentName, Age, Observation, TeacherNotes, ParentHopes.
│   │   ├── insight.go               # Response struct: TalentLabel, PersonalityAnalysis, Recommendations.
│   │   └── repository.go            # Interfaces for VectorDB and LLM interactions.
│   ├── handler/
│   │   └── insight_handler.go       # Gin HTTP handler: decodes JSON, calls service, returns standard JSON response.
│   ├── service/
│   │   └── rag_service.go           # Implements business logic: fetches RAG context, builds prompt, calls LLM.
│   └── repository/
│       ├── chroma_repo.go           # Concrete implementation for ChromaDB connection and vector search.
│       └── llm_repo.go              # Concrete implementation for LangChainGo Gemini interactions.
├── source_documents/                # (Do not generate files here, keep empty).
├── docker-compose.yml               # Config for chromadb/chroma image on port 8000.
├── go.mod
└── go.sum

```

**Step-by-Step Execution Plan for the AI Agent**

Please output the refactored code in the following order:

* **Config & Domain:** Start with internal/config/config.go and the structs/interfaces in internal/domain/.
* **Repositories:** Write internal/repository/chroma_repo.go and internal/repository/llm_repo.go. Ensure they implement the interfaces defined in the domain layer.
* **Service:** Write internal/service/rag_service.go. It must accept the repository interfaces via its constructor.
* **Handler:** Write internal/handler/insight_handler.go. It must parse HTTP requests and trigger the service.
* **Entry Points:** Write cmd/api/main.go (wiring everything together for the server) and cmd/ingester/main.go (the script to chunk and embed PDFs).
* **Infrastructure:** Provide the updated docker-compose.yml.

**Important Instruction:** Read the existing monolithic code first to understand the current LangChainGo implementation, then apply it strictly to this modular structure. Do not use placeholder comments for core logic; write the actual functional code.