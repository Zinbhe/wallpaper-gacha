package models

import (
	"database/sql"
	"time"
)

type User struct {
	DiscordID    string
	Username     string
	CreatedAt    time.Time
	LastUploadAt sql.NullTime
}

type Upload struct {
	ID               int
	DiscordID        string
	Filename         string
	OriginalFilename string
	FileSize         int64
	UploadedAt       time.Time
}

// GetOrCreateUser retrieves a user or creates one if it doesn't exist
func GetOrCreateUser(discordID, username string) (*User, error) {
	user := &User{}
	err := DB.QueryRow(
		"SELECT discord_id, username, created_at, last_upload_at FROM users WHERE discord_id = ?",
		discordID,
	).Scan(&user.DiscordID, &user.Username, &user.CreatedAt, &user.LastUploadAt)

	if err == sql.ErrNoRows {
		// Create new user
		_, err = DB.Exec(
			"INSERT INTO users (discord_id, username) VALUES (?, ?)",
			discordID, username,
		)
		if err != nil {
			return nil, err
		}
		return GetOrCreateUser(discordID, username)
	} else if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateLastUpload updates the last upload timestamp for a user
func (u *User) UpdateLastUpload() error {
	_, err := DB.Exec(
		"UPDATE users SET last_upload_at = CURRENT_TIMESTAMP WHERE discord_id = ?",
		u.DiscordID,
	)
	return err
}

// CanUpload checks if the user can upload based on the cooldown period
func (u *User) CanUpload(cooldownMinutes int) (bool, time.Duration) {
	if !u.LastUploadAt.Valid {
		return true, 0
	}

	nextUploadTime := u.LastUploadAt.Time.Add(time.Duration(cooldownMinutes) * time.Minute)
	now := time.Now()

	if now.Before(nextUploadTime) {
		return false, nextUploadTime.Sub(now)
	}

	return true, 0
}

// CreateUpload records a new upload in the database
func CreateUpload(discordID, filename, originalFilename string, fileSize int64) error {
	_, err := DB.Exec(
		"INSERT INTO uploads (discord_id, filename, original_filename, file_size) VALUES (?, ?, ?, ?)",
		discordID, filename, originalFilename, fileSize,
	)
	return err
}

// GetUserUploadCount returns the total number of uploads by a user
func GetUserUploadCount(discordID string) (int, error) {
	var count int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM uploads WHERE discord_id = ?",
		discordID,
	).Scan(&count)
	return count, err
}
