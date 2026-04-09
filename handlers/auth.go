package handlers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	clerkSDK "github.com/clerk/clerk-sdk-go/v2"
	clerkJWT "github.com/clerk/clerk-sdk-go/v2/jwt"
	clerkUser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/models"
)

type LoginRequest struct {
	Token string `json:"token" binding:"required"`
}

type LoginResponse struct {
	User         models.User `json:"user"`
	SessionToken string      `json:"session_token"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token diperlukan"})
		return
	}

	clerkSDK.SetKey(os.Getenv("CLERK_SECRET_KEY"))

	claims, err := clerkJWT.Verify(context.Background(), &clerkJWT.VerifyParams{
		Token: req.Token,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token Clerk tidak valid"})
		return
	}

	clerkUserID := claims.Subject

	clerkUserData, err := clerkUser.Get(context.Background(), clerkUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendapatkan data pengguna dari Clerk"})
		return
	}

	email := ""
	if len(clerkUserData.EmailAddresses) > 0 {
		email = clerkUserData.EmailAddresses[0].EmailAddress
	}

	firstName := ""
	if clerkUserData.FirstName != nil {
		firstName = *clerkUserData.FirstName
	}
	lastName := ""
	if clerkUserData.LastName != nil {
		lastName = *clerkUserData.LastName
	}
	name := strings.TrimSpace(firstName + " " + lastName)

	var user models.User
	result := db.DB.Where("clerk_id = ?", clerkUserID).First(&user)
	if result.Error != nil {
		user = models.User{
			ClerkID: clerkUserID,
			Email:   email,
			Name:    name,
		}
		if err := db.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun pengguna"})
			return
		}
	} else {
		user.Email = email
		user.Name = name
		db.DB.Save(&user)
	}

	sessionToken, err := generateSessionToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		User:         user,
		SessionToken: sessionToken,
	})
}

func generateSessionToken(userID uint) (string, error) {
	secretKey := os.Getenv("SECRET_KEY")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	return token.SignedString([]byte(secretKey))
}
