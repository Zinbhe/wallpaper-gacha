package handlers

import (
	"net/http"

	"github.com/Zinbhe/wallpaper-gacha/assets"
	"github.com/Zinbhe/wallpaper-gacha/middleware"
)

// HomeHandler serves the landing page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is already authenticated
	session, err := middleware.Store.Get(r, "wallpaper-session")
	if err == nil {
		if auth, ok := session.Values["authenticated"].(bool); ok && auth {
			http.Redirect(w, r, "/upload", http.StatusSeeOther)
			return
		}
	}

	content, err := assets.StaticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// UploadPageHandler serves the upload page
func UploadPageHandler(w http.ResponseWriter, r *http.Request) {
	content, err := assets.StaticFiles.ReadFile("static/upload.html")
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}
