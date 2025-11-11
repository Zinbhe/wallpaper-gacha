package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Zinbhe/wallpaper-gacha/config"
	"github.com/Zinbhe/wallpaper-gacha/handlers"
	"github.com/Zinbhe/wallpaper-gacha/middleware"
	"github.com/Zinbhe/wallpaper-gacha/models"
	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	configFile := "config.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	log.Printf("Loading configuration from %s", configFile)
	if err := config.Load(configFile); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	log.Printf("Initializing database at %s", config.AppConfig.DatabasePath)
	if err := models.InitDatabase(config.AppConfig.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer models.Close()

	// Initialize session store
	middleware.InitSessionStore(config.AppConfig.SessionSecret)

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(config.AppConfig.UploadDirectory, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Setup router
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/", handlers.HomeHandler).Methods("GET")
	r.HandleFunc("/auth/login", handlers.LoginHandler).Methods("GET")
	r.HandleFunc("/auth/callback", handlers.CallbackHandler).Methods("GET")
	r.HandleFunc("/auth/logout", handlers.LogoutHandler).Methods("GET")

	// Protected routes
	r.HandleFunc("/upload", middleware.RequireAuth(handlers.UploadPageHandler)).Methods("GET")
	r.HandleFunc("/api/upload", middleware.RequireAuth(handlers.UploadHandler)).Methods("POST")

	// Start server
	addr := fmt.Sprintf("%s:%d", config.AppConfig.ServerHost, config.AppConfig.ServerPort)
	log.Printf("Starting server on %s", addr)
	log.Printf("Upload cooldown: %d minutes", config.AppConfig.UploadCooldownMinutes)
	log.Printf("Max file size: %dMB", config.AppConfig.MaxFileSizeMB)
	log.Printf("Allowed Discord servers: %v", config.AppConfig.AllowedServerIDs)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
