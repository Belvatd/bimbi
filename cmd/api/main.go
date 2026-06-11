package main

import (
	"context"
	"fmt"
	"log"

	"bimbi-backend/internal/config"
	"bimbi-backend/internal/domain"
	"bimbi-backend/internal/handler"
	"bimbi-backend/internal/middleware"
	"bimbi-backend/internal/repository"
	"bimbi-backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Initialize Postgres Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to postgres: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	log.Println("✅ Connected to PostgreSQL database")

	// Alter action_feedbacks to allow longer activity names
	db.Exec("ALTER TABLE action_feedbacks ALTER COLUMN activity_name TYPE TEXT")

	// Auto-Migrate domain models
	if err := db.AutoMigrate(&domain.Child{}, &domain.Assessment{}, &domain.ActionFeedback{}); err != nil {
		log.Fatalf("FATAL: Failed to run auto-migration: %v", err)
	}

	// 2. Initialize Repositories
	userRepo := repository.NewPostgresUserRepo(db)
	childRepo := repository.NewPostgresChildRepo(db)
	assessmentRepo := repository.NewPostgresAssessmentRepo(db)
	feedbackRepo := repository.NewPostgresFeedbackRepo(db)
	
	vectorRepo := repository.NewChromaRepo(cfg.ChromaURL, cfg.GeminiKey)
	llmRepo, err := repository.NewLLMRepo(context.Background(), cfg.GeminiKey)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize LLM repo: %v", err)
	}

	// 3. Initialize Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	feedbackService := service.NewFeedbackService(feedbackRepo, assessmentRepo)
	ragService := service.NewRagService(vectorRepo, llmRepo, childRepo, assessmentRepo, feedbackRepo)

	// 4. Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	insightHandler := handler.NewInsightHandler(ragService)
	childHandler := handler.NewChildHandler(childRepo)
	assessmentHandler := handler.NewAssessmentHandler(assessmentRepo, childRepo)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)

	// 5. Setup Gin Router
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// ── Routes ────────────────────────────────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "bimbi-ai-backend",
			"version": "1.0.0",
		})
	})

	authRoutes := router.Group("/api/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/logout", authHandler.Logout)
	}

	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Children Routes
		protected.GET("/children", childHandler.GetChildren)
		protected.POST("/children", childHandler.CreateChild)
		protected.GET("/children/:id", childHandler.GetChild)
		protected.GET("/children/:id/dashboard", insightHandler.GetDashboard)
		protected.GET("/children/:id/home-activities", insightHandler.GetHomeActivities)
		protected.POST("/children/:id/feedback", feedbackHandler.SubmitFeedback)
		protected.PUT("/children/:id", childHandler.UpdateChild)
		protected.DELETE("/children/:id", childHandler.DeleteChild)

		// Assessment Routes
		protected.GET("/assessments", assessmentHandler.GetAssessments)
		protected.POST("/assessments", insightHandler.GenerateInsights)
		protected.GET("/assessments/:id", assessmentHandler.GetAssessment)
		protected.DELETE("/assessments/:id", assessmentHandler.DeleteAssessment)
	}

	log.Printf("🚀 Bimbi AI Backend started on http://localhost:%s", cfg.Port)
	log.Printf("📡 ChromaDB: %s", cfg.ChromaURL)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("FATAL: Server failed: %v", err)
	}
}
