# Prompt: Frontend Development for Bimbi AI

**Context & Objective**
* **Role:** Senior Frontend Engineer.
* **Task:** Build a Modern React Single Page Application (SPA) for **Bimbi AI**, an EdTech platform that tracks children's development using AI.
* **Tech Stack:** React 18+, TypeScript, React Router v7, Tailwind CSS (or styling framework of choice), Axios/Fetch for API requests, Context API/Zustand for state management.
* **Backend Context:** A fully functional Go/Gin RESTful backend is available at `http://localhost:8080`.

**App Structure & Routing (React Router v7)**
Implement the following route structure utilizing React Router v7's latest features (e.g., Layouts, Loaders, Actions):

1. **Public Routes (Auth Layout)**
   * `/login`: Login page. Integrates with `POST /api/auth/login`.
   * `/register`: Registration page. Integrates with `POST /api/auth/register`.

2. **Protected Routes (Main Layout - Requires JWT)**
   * `/`: Home Dashboard. Displays a list of the user's children (`GET /api/children`). Includes functionality to add a new child (`POST /api/children`).
   * `/children/:id`: Child's detailed profile. Contains sub-views or tabs:
     * **Timeline/Overview:** Displays the child's assessment history using the dashboard format (`GET /api/children/:id/dashboard`).
     * **Home Activities:** Shows recommended activities (`GET /api/children/:id/home-activities`). Allows parents to submit feedback (`POST /api/children/:id/feedback`) which updates the activity status.
     * **New Assessment:** A form to submit new child observations (`POST /api/assessments`).

**Key Features & Implementation Guidelines**

1. **Authentication & Security:**
   * Manage the JWT securely.
   * Implement a Protected Route wrapper or use Router Loaders to intercept unauthenticated requests and redirect to `/login`.
   * Configure a global API client (e.g., Axios interceptor) to automatically attach the `Authorization: Bearer <token>` header to all outgoing protected requests.

2. **Data Fetching & Type Safety:**
   * Define robust TypeScript interfaces reflecting the backend domain models (e.g., `Child`, `DashboardResponse`, `TimelineItem`, `AssessmentResponse`).
   * Use a data fetching library like React Query (TanStack Query) or SWR for caching, loading states, and optimistic UI updates, or utilize React Router v7's data routers.

3. **UI/UX Aesthetics:**
   * **Premium Design:** The interface must feel modern, empathetic, and engaging for parents. Use a curated color palette, soft gradients, and modern typography (e.g., Inter, Outfit).
   * **Micro-animations:** Incorporate smooth transitions for opening modals, submitting forms, and navigating between tabs.
   * **Responsive Layout:** Must be fully functional and beautiful on mobile screens.

4. **Forms & Validation:**
   * Use libraries like React Hook Form with Zod schema validation for robust form handling, especially for the complex "New Assessment" form.

**Step-by-Step Execution**

Please implement the solution following these phases:

* **Phase 1: Project Setup & Core Routing**
  * Initialize the project structure (Vite + React + TS).
  * Set up React Router v7 with a `RouterProvider`, defining the `AuthLayout` and `MainLayout` boundaries.
* **Phase 2: Authentication Flow**
  * Implement the API service for authentication.
  * Build the Login and Register forms with validation.
  * Implement the auth state context and protected route logic.
* **Phase 3: Children Management**
  * Build the main `/` view listing children.
  * Create the "Add Child" feature.
* **Phase 4: Detailed Insight View & Assessment**
  * Implement the `/children/:id` layout.
  * Build the Assessment Timeline view.
  * Build the Home Activities view and Feedback submission.
  * Build the Assessment input form.

**Output Generation**
Please start by providing the initial project structure, the React Router v7 configuration (`App.tsx` or router definition file), and the Authentication Context/State setup.
