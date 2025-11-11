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
		log.Printf("Invalid upload attempt with method %s from IP: %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	discordID := middleware.GetDiscordID(r)
	username := middleware.GetUsername(r)

	if discordID == "" {
		log.Printf("Upload attempt without authentication from IP: %s", r.RemoteAddr)
		respondJSON(w, http.StatusUnauthorized, UploadResponse{
			Success: false,
			Message: "Not authenticated",
		})
		return
	}

	log.Printf("Upload attempt by user %s (ID: %s) from IP: %s", username, discordID, r.RemoteAddr)

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
		log.Printf("Upload denied for user %s (ID: %s): rate limit exceeded, cooldown: %v", username, discordID, cooldown)
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
		log.Printf("Upload failed for user %s (ID: %s): file too large (max %dMB)", username, discordID, config.AppConfig.MaxFileSizeMB)
		respondJSON(w, http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: fmt.Sprintf("File too large (max %dMB)", config.AppConfig.MaxFileSizeMB),
		})
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("wallpaper")
	if err != nil {
		log.Printf("Upload failed for user %s (ID: %s): no file provided - %v", username, discordID, err)
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
		log.Printf("Upload failed for user %s (ID: %s): invalid file extension '%s' for file '%s'", username, discordID, ext, header.Filename)
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
		log.Printf("Upload failed for user %s (ID: %s): failed to read file '%s' - %v", username, discordID, header.Filename, err)
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
		log.Printf("Upload failed for user %s (ID: %s): invalid MIME type '%s' for file '%s'", username, discordID, contentType, header.Filename)
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
		log.Printf("Upload failed for user %s (ID: %s): failed to create upload directory - %v", username, discordID, err)
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
		log.Printf("Upload failed for user %s (ID: %s): failed to create destination file - %v", username, discordID, err)
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
		log.Printf("Upload failed for user %s (ID: %s): failed to copy file - %v", username, discordID, err)
		os.Remove(destPath) // Clean up partial file
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to save file",
		})
		return
	}

	// Record upload in database
	if err := models.CreateUpload(discordID, newFilename, header.Filename, written); err != nil {
		log.Printf("Upload failed for user %s (ID: %s): failed to record upload in database - %v", username, discordID, err)
		os.Remove(destPath) // Clean up file since DB record failed
		respondJSON(w, http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to record upload",
		})
		return
	}

	// Update user's last upload time
	if err := user.UpdateLastUpload(); err != nil {
		log.Printf("Warning: Failed to update last upload time for user %s (ID: %s): %v", username, discordID, err)
	}

	// Get total upload count
	uploadCount, _ := models.GetUserUploadCount(discordID)

	log.Printf("Upload successful: user %s (ID: %s) uploaded '%s' as '%s', size: %d bytes, total uploads: %d",
		username, discordID, header.Filename, newFilename, written, uploadCount)

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
