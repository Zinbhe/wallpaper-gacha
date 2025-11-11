package handlers

import (
	"net/http"
	"path/filepath"

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

	http.ServeFile(w, r, filepath.Join("static", "index.html"))
}

// UploadPageHandler serves the upload page
func UploadPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("static", "upload.html"))
}
