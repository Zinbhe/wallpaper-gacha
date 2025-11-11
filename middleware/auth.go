package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
)

type contextKey string

const (
	DiscordIDKey contextKey = "discord_id"
	UsernameKey  contextKey = "username"
)

var Store *sessions.CookieStore

// InitSessionStore initializes the session store with a secret key
func InitSessionStore(secret string) {
	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   true, // Only send cookie over HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

// RequireAuth is middleware that requires a valid session
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := Store.Get(r, "wallpaper-session")
		if err != nil {
			// Invalid/stale session cookie - redirect to login (new login will overwrite with valid cookie)
			log.Printf("Authentication required: invalid session cookie for %s %s from IP: %s: %v", r.Method, r.URL.Path, r.RemoteAddr, err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		auth, ok := session.Values["authenticated"].(bool)
		if !ok || !auth {
			log.Printf("Authentication required: unauthenticated access attempt to %s %s from IP: %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		discordID, ok := session.Values["discord_id"].(string)
		if !ok {
			log.Printf("Authentication required: missing discord_id for %s %s from IP: %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		username, ok := session.Values["username"].(string)
		if !ok {
			username = "Unknown"
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), DiscordIDKey, discordID)
		ctx = context.WithValue(ctx, UsernameKey, username)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetDiscordID retrieves the Discord ID from request context
func GetDiscordID(r *http.Request) string {
	if discordID, ok := r.Context().Value(DiscordIDKey).(string); ok {
		return discordID
	}
	return ""
}

// GetUsername retrieves the username from request context
func GetUsername(r *http.Request) string {
	if username, ok := r.Context().Value(UsernameKey).(string); ok {
		return username
	}
	return ""
}
