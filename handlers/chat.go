package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/db"
	"github.com/ridwanFatur/hukumai-project-backend/models"
	appws "github.com/ridwanFatur/hukumai-project-backend/ws"
)

type wsMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// POST /api/chat/sessions
func CreateChatSession(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	session := models.ChatSession{
		UserID: user.ID,
		Title:  "Sesi Chat Baru",
	}
	if err := db.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi chat"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": session})
}

// GET /api/chat/sessions
func GetChatSessions(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	var sessions []models.ChatSession
	db.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&sessions)
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GET /api/chat/sessions/:id/messages
func GetChatMessages(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID sesi tidak valid"})
		return
	}

	var session models.ChatSession
	if err := db.DB.Where("id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi chat tidak ditemukan"})
		return
	}

	var messages []models.Chat
	db.DB.Where("chat_session_id = ?", sessionID).Order("created_at ASC").Find(&messages)
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// POST /api/chat/sessions/:id/messages
// Saves the user message, then triggers background AI processing.
func SendMessage(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID sesi tidak valid"})
		return
	}

	var session models.ChatSession
	if err := db.DB.Where("id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi chat tidak ditemukan"})
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan tidak boleh kosong"})
		return
	}

	userMsg := models.Chat{
		ChatSessionID: uint(sessionID),
		Role:          "user",
		Content:       req.Content,
	}
	if err := db.DB.Create(&userMsg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pesan"})
		return
	}

	// Hand off to a goroutine — response will arrive via WebSocket
	go generateAIResponse(user.ID, uint(sessionID))

	c.JSON(http.StatusAccepted, gin.H{"message": userMsg})
}

// generateAIResponse runs in a goroutine, waits, then pushes the reply
// through the WebSocket hub.
func generateAIResponse(userID uint, sessionID uint) {
	time.Sleep(5 * time.Second)

	// TODO: Replace dummy reply with a real AI model call
	reply := "Terima kasih atas pertanyaan Anda. Ini adalah respons sementara dari sistem AI hukum kami. Fitur AI sesungguhnya akan segera tersedia."

	aiMsg := models.Chat{
		ChatSessionID: sessionID,
		Role:          "assistant",
		Content:       reply,
	}
	if err := db.DB.Create(&aiMsg).Error; err != nil {
		return
	}

	appws.GlobalHub.SendJSON(userID, wsMessage{ //nolint:errcheck
		Type: "chat_message",
		Data: aiMsg,
	})
}
