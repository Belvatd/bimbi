# Objective
Create a secure and scalable authentication system using Go and PostgreSQL running in Docker.

# Requirements

## 1. Database Setup (Docker)
- Create a `docker-compose.yml` file to run PostgreSQL.
- Ensure the database configuration uses environment variables for security (avoid hardcoding credentials).
- Include a persistent volume for the database data so it survives container restarts.

## 2. Authentication Flow
- Implement user registration (signup) and user login endpoints.
- Use secure password hashing (e.g., `bcrypt` or `argon2`). Never store plain text passwords.
- Implement JWT (JSON Web Tokens) or secure, HTTP-only session cookies for maintaining the user's authenticated state.
- Create an authentication middleware to protect private API routes.

## 3. Go Implementation
- Use standard Go best practices, such as the repository pattern or clean architecture.
- Use a robust database driver like `pgx` (recommended) or an ORM like `gorm`.
- Implement robust database connection pooling.
- Ensure proper error handling and ensure sensitive information (like DB errors) is not leaked in API responses.

## 4. Best Practices & Security
- Validate all incoming user inputs (e.g., email format, password strength).
- Protect against SQL injection by exclusively using parameterized queries.
- Ensure passwords are never logged or returned in HTTP responses.
- Store sensitive configuration (DB credentials, JWT secrets) in a `.env` file and load them securely using a library like `godotenv`.
- (Bonus) Consider rate limiting on the login endpoint to mitigate brute-force attacks.

# Output Expectations
- Please provide the `docker-compose.yml` file.
- Provide the SQL schema or migration scripts for the `users` table.
- Provide the Go code for:
  - Database connection setup.
  - User models and repository interface.
  - Authentication handlers (Register & Login).
  - Auth middleware.
- Briefly explain the security measures and architectural choices implemented.
