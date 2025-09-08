package api

import (
	"llm-inferra/internal/api/handlers"
	"llm-inferra/internal/api/middleware"
	"llm-inferra/internal/config"
	"llm-inferra/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	db     *gorm.DB
	config *config.Config
	router *gin.Engine
}

func NewServer(db *gorm.DB, cfg *config.Config) *Server {
	server := &Server{
		db:     db,
		config: cfg,
	}

	server.setupRouter()
	return server
}

func (s *Server) setupRouter() {
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = s.config.CORSOrigins
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	s.router.Use(cors.New(corsConfig))

	// Rate limiting
	s.router.Use(middleware.RateLimit(s.config.RateLimitRPS))

	// Initialize services
	authService := services.NewAuthService(s.db, s.config)
	userService := services.NewUserService(s.db)
	providerService := services.NewProviderService(s.db)
	apiKeyService := services.NewAPIKeyService(s.db)
	analyticsService := services.NewAnalyticsService(s.db)
	llmService := services.NewLLMService(s.db, nil, apiKeyService, providerService, analyticsService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService)
	userHandler := handlers.NewUserHandler(userService)
	providerHandler := handlers.NewProviderHandler(providerService)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	llmHandler := handlers.NewLLMHandler(llmService)

	// Public routes
	public := s.router.Group("/api/v1")
	{
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/register", authHandler.Register)
		public.GET("/health", s.healthCheck)
	}

	// LLM API routes (OpenAI-compatible endpoints)
	llmAPI := s.router.Group("/v1")
	{
		llmAPI.POST("/chat/completions", llmHandler.ChatCompletion)
		llmAPI.GET("/models", llmHandler.ListModels)
		llmAPI.GET("/health", llmHandler.HealthCheck)
	}

	// Protected routes
	protected := s.router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(s.config.JWTSecret))
	{
		// User management
		users := protected.Group("/users")
		{
			users.GET("", middleware.PaginationMiddleware(), userHandler.ListUsers)
			users.GET("/me", userHandler.GetCurrentUser)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}

		// Provider management
		providers := protected.Group("/providers")
		{
			providers.GET("", middleware.PaginationMiddleware(), providerHandler.ListProviders)
			providers.POST("", providerHandler.CreateProvider)
			providers.GET("/:id", providerHandler.GetProvider)
			providers.PUT("/:id", providerHandler.UpdateProvider)
			providers.DELETE("/:id", providerHandler.DeleteProvider)
			providers.GET("/:id/models", middleware.PaginationMiddleware(), providerHandler.ListModels)
			providers.POST("/:id/models", providerHandler.CreateModel)
		}

		// Model management
		models := protected.Group("/models")
		{
			models.GET("", middleware.PaginationMiddleware(), providerHandler.ListAllModels)
			models.GET("/:id", providerHandler.GetModel)
			models.PUT("/:id", providerHandler.UpdateModel)
			models.DELETE("/:id", providerHandler.DeleteModel)
		}

		// API Key management
		apiKeys := protected.Group("/api-keys")
		{
			apiKeys.GET("", middleware.PaginationMiddleware(), apiKeyHandler.ListAPIKeys)
			apiKeys.POST("", apiKeyHandler.CreateAPIKey)
			apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
			apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
			apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
		}

		// Analytics and monitoring
		analytics := protected.Group("/analytics")
		{
			analytics.GET("/overview", analyticsHandler.GetOverview)
			analytics.GET("/usage", middleware.PaginationMiddleware(), analyticsHandler.GetUsageAnalytics)
			analytics.GET("/costs", analyticsHandler.GetCostAnalytics)
			analytics.GET("/users", middleware.PaginationMiddleware(), analyticsHandler.GetUserAnalytics)
			analytics.GET("/providers", analyticsHandler.GetProviderAnalytics)
			analytics.GET("/models", analyticsHandler.GetModelAnalytics)
		}

		// System health (admin only)
		system := protected.Group("/system")
		system.Use(middleware.AdminMiddleware())
		{
			system.GET("/health", analyticsHandler.GetSystemHealth)
			system.GET("/logs", middleware.PaginationMiddleware(), analyticsHandler.GetLogs)
		}
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "llm-inferra",
		"version": "1.0.0",
	})
}

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
