package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/models"
	appws "github.com/ridwanFatur/hukumai-project-backend/ws"
)

var upgrader = websocket.Upgrader{
	// Origin check is intentionally permissive here; the backend CORS
	// middleware governs HTTP requests, but WebSocket upgrade bypasses it.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketConnect handles GET /ws.
// The client must pass its backend session token as the `token` query param.
func WebSocketConnect(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		// Also accept Authorization header for symmetry
		tokenStr = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak ditemukan"})
		return
	}

	secretKey := os.Getenv("SECRET_KEY")
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secretKey), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		return
	}
	rawID, ok := claims["sub"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, uint(rawID)).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Pengguna tidak ditemukan"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	appws.GlobalHub.Register(user.ID, conn)
	defer appws.GlobalHub.Unregister(user.ID)

	// Read loop — keeps the connection alive and handles client-initiated close.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
