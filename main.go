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
	db.DB.AutoMigrate(&models.User{})

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

	api := r.Group("/api")
	{
		api.POST("/auth/login", handlers.Login)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/users/me", handlers.GetMe)
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
