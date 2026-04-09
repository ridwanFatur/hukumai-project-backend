package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanFatur/hukumai-project-backend/models"
)

// allowedMIMETypes maps accepted MIME types to a human-readable label.
var allowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
	"application/msword": true, // .doc
}

const maxUploadSize = 50 << 20 // 50 MB

// UploadDocument receives a multipart file, validates it, uploads it to
// Supabase Storage, and returns the public URL and storage key.
//
// Required env vars:
//   SUPABASE_URL            – e.g. https://<project>.supabase.co
//   SUPABASE_ANON_KEY       – anon/service key
//   SUPABASE_STORAGE_BUCKET – target bucket name (e.g. "documents")
func UploadDocument(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	// Enforce max body size before parsing
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Ukuran file melebihi batas maksimum 50MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan dalam permintaan"})
		return
	}
	defer file.Close()

	// Read file content (needed for MIME detection and upload)
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}

	// Detect MIME type from the first 512 bytes
	mimeType := http.DetectContentType(content)
	// http.DetectContentType only reads magic bytes; fall back to the
	// Content-Type header for DOCX/DOC which look like generic binary data.
	if mimeType == "application/octet-stream" || mimeType == "application/zip" {
		if ct := header.Header.Get("Content-Type"); ct != "" {
			mimeType = ct
		}
	}

	if !allowedMIMETypes[mimeType] {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "Tipe file tidak didukung. Silakan unggah file PDF atau DOCX.",
		})
		return
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	bucket := os.Getenv("SUPABASE_STORAGE_BUCKET")

	if supabaseURL == "" || anonKey == "" || bucket == "" {
		// TODO: Configure SUPABASE_URL, SUPABASE_ANON_KEY, SUPABASE_STORAGE_BUCKET in .env
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Layanan penyimpanan belum dikonfigurasi"})
		return
	}

	// Build a unique storage path: documents/{userID}/{timestamp}_{filename}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	safeFilename := sanitizeFilename(strings.TrimSuffix(header.Filename, ext))
	storagePath := fmt.Sprintf("documents/%d/%d_%s%s",
		user.ID,
		time.Now().UnixMilli(),
		safeFilename,
		ext,
	)

	publicURL, err := uploadToSupabase(supabaseURL, anonKey, bucket, storagePath, mimeType, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah file ke penyimpanan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_key":   bucket + "/" + storagePath,
		"public_url": publicURL,
		"filename":   header.Filename,
		"size":       header.Size,
		"mime_type":  mimeType,
	})
}

// uploadToSupabase uploads content to Supabase Storage via the REST API and
// returns the public URL.
func uploadToSupabase(supabaseURL, anonKey, bucket, path, mimeType string, content []byte) (string, error) {
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, path)

	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+anonKey)
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, bucket, path)
	return publicURL, nil
}

// sanitizeFilename removes characters unsafe for storage paths.
func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
