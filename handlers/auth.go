package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Zinbhe/wallpaper-gacha/config"
	"github.com/Zinbhe/wallpaper-gacha/middleware"
	"github.com/Zinbhe/wallpaper-gacha/models"
	"github.com/gorilla/sessions"
)

type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type DiscordGuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// LoginHandler redirects to Discord OAuth
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("User initiated Discord OAuth authentication from IP: %s", r.RemoteAddr)
	authURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify%%20guilds",
		config.AppConfig.DiscordClientID,
		url.QueryEscape(config.AppConfig.DiscordRedirectURI),
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth callback from Discord
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("OAuth callback failed: no code provided from IP: %s", r.RemoteAddr)
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	log.Printf("Processing OAuth callback from IP: %s", r.RemoteAddr)

	// Exchange code for access token
	token, err := exchangeCode(code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Failed to authenticate with Discord", http.StatusInternalServerError)
		return
	}

	// Get user info
	user, err := getDiscordUser(token)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Get user's guilds
	guilds, err := getDiscordGuilds(token)
	if err != nil {
		log.Printf("Failed to get guilds: %v", err)
		http.Error(w, "Failed to verify server membership", http.StatusInternalServerError)
		return
	}

	// Check if user is in an allowed server
	if !isInAllowedServer(guilds) {
		log.Printf("Authentication denied: user %s (ID: %s) not in allowed Discord servers", user.Username, user.ID)
		http.Error(w, "You are not in an allowed Discord server", http.StatusForbidden)
		return
	}

	log.Printf("User %s (ID: %s) verified in allowed Discord server", user.Username, user.ID)

	// Create or update user in database
	dbUser, err := models.GetOrCreateUser(user.ID, user.Username)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create session - if there's an invalid/stale cookie, create a new session
	session, err := middleware.Store.Get(r, "wallpaper-session")
	if err != nil {
		log.Printf("Invalid session cookie detected for user %s (ID: %s), creating new session: %v", user.Username, user.ID, err)
		// Create a fresh session using sessions.NewSession
		session = sessions.NewSession(middleware.Store, "wallpaper-session")
	}

	session.Values["discord_id"] = dbUser.DiscordID
	session.Values["username"] = dbUser.Username
	session.Values["authenticated"] = true

	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to save session for user %s (ID: %s): %v", dbUser.Username, dbUser.DiscordID, err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	log.Printf("User successfully authenticated: %s (ID: %s) from IP: %s", dbUser.Username, dbUser.DiscordID, r.RemoteAddr)
	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

// LogoutHandler destroys the session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := middleware.Store.Get(r, "wallpaper-session")
	if err != nil {
		log.Printf("Logout attempt with invalid session from IP: %s", r.RemoteAddr)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get user info before destroying session
	username, _ := session.Values["username"].(string)
	discordID, _ := session.Values["discord_id"].(string)

	session.Options.MaxAge = -1
	session.Save(r, w)

	if username != "" && discordID != "" {
		log.Printf("User logged out: %s (ID: %s) from IP: %s", username, discordID, r.RemoteAddr)
	} else {
		log.Printf("User logged out from IP: %s", r.RemoteAddr)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func exchangeCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", config.AppConfig.DiscordClientID)
	data.Set("client_secret", config.AppConfig.DiscordClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.AppConfig.DiscordRedirectURI)

	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func getDiscordUser(token string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user: %s", string(body))
	}

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func getDiscordGuilds(token string) ([]DiscordGuild, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get guilds: %s", string(body))
	}

	var guilds []DiscordGuild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, err
	}

	return guilds, nil
}

func isInAllowedServer(guilds []DiscordGuild) bool {
	allowedServers := make(map[string]bool)
	for _, id := range config.AppConfig.AllowedServerIDs {
		allowedServers[id] = true
	}

	for _, guild := range guilds {
		if allowedServers[guild.ID] {
			return true
		}
	}

	return false
}

// UserInfoHandler returns the current user's information
func UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsername(r)
	discordID := middleware.GetDiscordID(r)

	if discordID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username":   username,
		"discord_id": discordID,
	})
}
