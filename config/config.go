package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	ServerPort             int      `json:"server_port"`
	ServerHost             string   `json:"server_host"`
	DiscordClientID        string   `json:"discord_client_id"`
	DiscordClientSecret    string   `json:"discord_client_secret"`
	DiscordRedirectURI     string   `json:"discord_redirect_uri"`
	AllowedServerIDs       []string `json:"allowed_server_ids"`
	UploadCooldownMinutes  int      `json:"upload_cooldown_minutes"`
	MaxFileSizeMB          int      `json:"max_file_size_mb"`
	DatabasePath           string   `json:"database_path"`
	UploadDirectory        string   `json:"upload_directory"`
	SessionSecret          string   `json:"session_secret"`
}

var AppConfig *Config

// Load reads and parses the configuration file
func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	AppConfig = &Config{}
	if err := decoder.Decode(AppConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if AppConfig.DiscordClientID == "" {
		return fmt.Errorf("discord_client_id is required")
	}
	if AppConfig.DiscordClientSecret == "" {
		return fmt.Errorf("discord_client_secret is required")
	}
	if AppConfig.DiscordRedirectURI == "" {
		return fmt.Errorf("discord_redirect_uri is required")
	}
	if len(AppConfig.AllowedServerIDs) == 0 {
		return fmt.Errorf("at least one allowed_server_id is required")
	}
	if AppConfig.SessionSecret == "" {
		return fmt.Errorf("session_secret is required")
	}

	// Set defaults
	if AppConfig.ServerPort == 0 {
		AppConfig.ServerPort = 8080
	}
	if AppConfig.ServerHost == "" {
		AppConfig.ServerHost = "localhost"
	}
	if AppConfig.UploadCooldownMinutes == 0 {
		AppConfig.UploadCooldownMinutes = 60
	}
	if AppConfig.MaxFileSizeMB == 0 {
		AppConfig.MaxFileSizeMB = 50
	}
	if AppConfig.DatabasePath == "" {
		AppConfig.DatabasePath = "./wallpaper.db"
	}
	if AppConfig.UploadDirectory == "" {
		AppConfig.UploadDirectory = "./uploads"
	}

	return nil
}
