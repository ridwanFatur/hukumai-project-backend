package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/models"
)

func GetMe(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user.(models.User)})
}
