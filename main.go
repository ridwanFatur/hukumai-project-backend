package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/models"
)

func main() {
	db.Connect()

	db.DB.AutoMigrate(&models.User{})

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.Run()
}
