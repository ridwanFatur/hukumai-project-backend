package main

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/handlers"
	"github.com/ridwanFatur/hukumai-project-backend/middleware"
	"github.com/ridwanFatur/hukumai-project-backend/models"
)

func main() {
	db.Connect()
	db.DB.AutoMigrate(&models.User{}, &models.ChatSession{}, &models.Chat{}, &models.SubscriptionPlan{}, &models.Subscription{})
	db.SeedSubscriptionPlans()

	r := gin.Default()

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000"
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     splitOrigins(corsOrigins),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// WebSocket endpoint — auth is handled inside the handler via query param
	r.GET("/ws", handlers.WebSocketConnect)

	api := r.Group("/api")
	{
		api.POST("/auth/login", handlers.Login)
		api.GET("/subscriptions/plans", handlers.GetPlans)
		api.POST("/subscriptions/webhook", handlers.StripeWebhook)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/users/me", handlers.GetMe)

			protected.POST("/chat/sessions", handlers.CreateChatSession)
			protected.GET("/chat/sessions", handlers.GetChatSessions)
			protected.GET("/chat/sessions/:id/messages", handlers.GetChatMessages)
			protected.POST("/chat/sessions/:id/messages", handlers.SendMessage)

			protected.GET("/subscriptions/status", handlers.GetSubscriptionStatus)
			protected.POST("/subscriptions/checkout", handlers.CreateCheckout)

			protected.POST("/upload", handlers.UploadDocument)
		}
	}

	r.Run()
}

func splitOrigins(origins string) []string {
	var result []string
	start := 0
	for i, c := range origins {
		if c == ',' {
			result = append(result, origins[start:i])
			start = i + 1
		}
	}
	result = append(result, origins[start:])
	return result
}
