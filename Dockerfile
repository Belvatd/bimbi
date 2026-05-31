# Multi-stage build for Go application
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/api ./cmd/api/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for external API requests (e.g. Google Gemini API)
RUN apk --no-cache add ca-certificates tzdata

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/api .

# Expose the API port
EXPOSE 8080

# Command to run the application
CMD ["./api"]
