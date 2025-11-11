package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zinbhe/wallpaper-gacha/config"
	"github.com/Zinbhe/wallpaper-gacha/middleware"
	"github.com/Zinbhe/wallpaper-gacha/models"
	"github.com/google/uuid"
)

var allowedExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".jxl":  true,
	".webp": true,
}

var allowedMimeTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jxl":  true,
	"image/webp": true,
}

type UploadResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Filename     string `json:"filename,omitempty"`
	UploadCount  int    `json:"upload_count,omitempty"`
	CooldownSecs int    `json:"cooldown_seconds,omitempty"`
}

// UploadHandler handles image uploads
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	discordID := middleware.GetDiscordID(r)
	if discordID == "" {
		respondJSON(w, http.StatusUnauthorized, UploadResponse{
			Success: false,
			Message: "Not authenticated",
		})
		return
	}

	// Get user from database
	user, err := models.GetOrCreateUser(discordID, middleware.GetUsername(r))
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to get user information",
		})
		return
	}

	// Check rate limit
	canUpload, cooldown := user.CanUpload(config.AppConfig.UploadCooldownMinutes)
	if !canUpload {
		respondJSON(w, http.StatusTooManyRequests, UploadResponse{
			Success:      false,
			Message:      fmt.Sprintf("Please wait %s before uploading again", formatDuration(cooldown)),
			CooldownSecs: int(cooldown.Seconds()),
		})
		return
	}

	// Parse multipart form with max memory
	maxSize := int64(config.AppConfig.MaxFileSizeMB * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondJSON(w, http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: fmt.Sprintf("File too large (max %dMB)", config.AppConfig.MaxFileSizeMB),
		})
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("wallpaper")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "No file provided",
		})
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		respondJSON(w, http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Invalid file type. Allowed: png, jpg, jpeg, jxl, webp",
		})
		return
	}

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to read file",
		})
		return
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Validate MIME type
	contentType := http.DetectContentType(buffer)
	// JXL might not be detected properly, so we allow it if extension is .jxl
	if !allowedMimeTypes[contentType] && ext != ".jxl" {
		respondJSON(w, http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Invalid file content type",
		})
		return
	}

	// Generate unique filename
	uniqueID := uuid.New().String()
	newFilename := uniqueID + ext

	// Create upload directory if it doesn't exist
	uploadDir := config.AppConfig.UploadDirectory
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Failed to create upload directory: %v", err)
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to create upload directory",
		})
		return
	}

	// Save the file
	destPath := filepath.Join(uploadDir, newFilename)
	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create destination file: %v", err)
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to save file",
		})
		return
	}
	defer destFile.Close()

	// Copy file contents
	written, err := io.Copy(destFile, file)
	if err != nil {
		log.Printf("Failed to copy file: %v", err)
		os.Remove(destPath) // Clean up partial file
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to save file",
		})
		return
	}

	// Record upload in database
	if err := models.CreateUpload(discordID, newFilename, header.Filename, written); err != nil {
		log.Printf("Failed to record upload: %v", err)
		os.Remove(destPath) // Clean up file since DB record failed
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to record upload",
		})
		return
	}

	// Update user's last upload time
	if err := user.UpdateLastUpload(); err != nil {
		log.Printf("Failed to update last upload time: %v", err)
	}

	// Get total upload count
	uploadCount, _ := models.GetUserUploadCount(discordID)

	respondJSON(w, http.StatusOK, UploadResponse{
		Success:     true,
		Message:     "Upload successful!",
		Filename:    newFilename,
		UploadCount: uploadCount,
	})
}

func respondJSON(w http.ResponseWriter, status int, data UploadResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func formatDuration(d interface{}) string {
	switch v := d.(type) {
	case int:
		if v < 60 {
			return fmt.Sprintf("%d seconds", v)
		}
		return fmt.Sprintf("%d minutes", v/60)
	default:
		// Handle time.Duration
		return fmt.Sprintf("%.0f minutes", d.(interface{ Minutes() float64 }).Minutes())
	}
}
