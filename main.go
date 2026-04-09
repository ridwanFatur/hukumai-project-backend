package main

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/handlers"
	"github.com/ridwanFatur/hukumai-project-backend/middleware"
	"github.com/ridwanFatur/hukumai-project-backend/models"
	"golang.org/x/time/rate"
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
		// Login: 10 req/min per IP to prevent brute-force.
		// TODO: adjust RATE_LOGIN_RPM in .env to tune (e.g. RATE_LOGIN_RPM=20 for higher traffic)
		api.POST("/auth/login",
			middleware.RateLimitIP(rate.Every(time.Minute)/10, 5, "RATE_LOGIN_RPM"),
			handlers.Login,
		)

		// Subscription plans are public and rarely change — light rate limit to
		// prevent scraping. Cache on the frontend avoids most repeat calls.
		// TODO: adjust RATE_PLANS_RPM in .env (default: 30/min per IP)
		api.GET("/subscriptions/plans",
			middleware.RateLimitIP(rate.Every(time.Minute)/30, 10, "RATE_PLANS_RPM"),
			handlers.GetPlans,
		)

		// Stripe webhook: no rate limit — Stripe retries and we must not drop events.
		api.POST("/subscriptions/webhook", handlers.StripeWebhook)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// User profile: light limit — data rarely changes and frontend caches it.
			// TODO: adjust RATE_USER_RPM in .env (default: 60/min per user)
			protected.GET("/users/me",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_USER_RPM"),
				handlers.GetMe,
			)

			// Chat session creation: moderate limit to prevent session spam.
			// TODO: adjust RATE_SESSION_CREATE_RPM in .env (default: 10/min per user)
			protected.POST("/chat/sessions",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_SESSION_CREATE_RPM"),
				handlers.CreateChatSession,
			)

			// Listing sessions: heavier read, short-term cached on frontend.
			// TODO: adjust RATE_SESSION_LIST_RPM in .env (default: 30/min per user)
			protected.GET("/chat/sessions",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_SESSION_LIST_RPM"),
				handlers.GetChatSessions,
			)

			// Message history: similar to session list.
			// TODO: adjust RATE_MESSAGES_RPM in .env (default: 30/min per user)
			protected.GET("/chat/sessions/:id/messages",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_MESSAGES_RPM"),
				handlers.GetChatMessages,
			)

			// Sending a message triggers AI processing — most resource-intensive endpoint.
			// TODO: adjust RATE_CHAT_SEND_RPM in .env (default: 20/min per user)
			protected.POST("/chat/sessions/:id/messages",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_CHAT_SEND_RPM"),
				handlers.SendMessage,
			)

			// Subscription status: cached on frontend; low limit acceptable.
			// TODO: adjust RATE_SUB_STATUS_RPM in .env (default: 20/min per user)
			protected.GET("/subscriptions/status",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_SUB_STATUS_RPM"),
				handlers.GetSubscriptionStatus,
			)

			// Checkout: payment action — very low limit to prevent abuse.
			// TODO: adjust RATE_CHECKOUT_RPM in .env (default: 5/min per user)
			protected.POST("/subscriptions/checkout",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_CHECKOUT_RPM"),
				handlers.CreateCheckout,
			)

			// Upload: bandwidth/storage intensive — strict per-user limit.
			// TODO: adjust RATE_UPLOAD_RPM in .env (default: 5/min per user)
			protected.POST("/upload",
				middleware.RateLimitUser(rate.Every(time.Minute)/60, 60, "RATE_UPLOAD_RPM"),
				handlers.UploadDocument,
			)
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
