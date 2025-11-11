# 🎨 Wallpaper Gacha

A fun web application that allows friends to upload and share wallpapers through Discord OAuth authentication.

## Features

- Discord OAuth2 authentication
- Server membership verification (whitelist specific Discord servers)
- Image upload with validation (PNG, JPG, JPEG, JPEG XL, WebP)
- Rate limiting (1 upload per hour, configurable)
- SQLite database for user and upload tracking
- Clean, modern web interface
- Support for large 4K wallpapers (up to 50MB)

## Prerequisites

- Go 1.21 or higher
- GCC compiler (required for CGo and SQLite)
- A Discord application (see setup below)
- A domain with HTTPS (for OAuth callback)
- Caddy or another reverse proxy for HTTPS

## Discord Application Setup

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name
3. Go to the "OAuth2" section
4. Add a redirect URL: `https://yourdomain.com/auth/callback`
5. Under "OAuth2 URL Generator":
   - Select scopes: `identify` and `guilds`
   - Copy the Client ID and Client Secret

**Note:** You don't need to create a bot for this application.

## Installation

1. Clone the repository:
```bash
git clone https://github.com/Zinbhe/wallpaper-gacha.git
cd wallpaper-gacha
```

2. Install dependencies:
```bash
go mod download
```

3. Create a configuration file:
```bash
cp config.example.json config.json
```

4. Edit `config.json` with your settings:
```json
{
  "server_port": 8080,
  "server_host": "localhost",
  "discord_client_id": "YOUR_CLIENT_ID",
  "discord_client_secret": "YOUR_CLIENT_SECRET",
  "discord_redirect_uri": "https://yourdomain.com/auth/callback",
  "allowed_server_ids": ["YOUR_SERVER_ID"],
  "upload_cooldown_minutes": 60,
  "max_file_size_mb": 50,
  "database_path": "./wallpaper.db",
  "upload_directory": "./uploads",
  "session_secret": "GENERATE_A_RANDOM_SECRET_HERE"
}
```

## Getting Your Discord Server ID

1. Enable Developer Mode in Discord:
   - User Settings → Advanced → Developer Mode (toggle on)
2. Right-click on your server icon
3. Click "Copy Server ID"
4. Paste the ID into the `allowed_server_ids` array in config.json

## Generating a Session Secret

Generate a secure random string for your session secret:

```bash
openssl rand -base64 32
```

Or in Go:
```bash
go run -c 'import("crypto/rand");import("encoding/base64");b:=make([]byte,32);rand.Read(b);println(base64.StdEncoding.EncodeToString(b))'
```

## Building

**Important:** CGo must be enabled for compilation (required for SQLite driver).

Build the application:
```bash
CGO_ENABLED=1 go build -o wallpaper-gacha
```

Or for a smaller binary:
```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o wallpaper-gacha
```

**Note:** CGo is enabled by default on most systems, but it's explicitly set here to ensure proper compilation. If you encounter build errors related to SQLite, make sure you have a C compiler (GCC) installed.

## Running

Run the application:
```bash
./wallpaper-gacha
```

Or specify a custom config file:
```bash
./wallpaper-gacha /path/to/config.json
```

The application will:
- Create the database file if it doesn't exist
- Create the uploads directory if it doesn't exist
- Start listening on the configured host and port

## Caddy Configuration

Here's an example Caddyfile for reverse proxying:

```caddy
yourdomain.com {
    reverse_proxy localhost:8080

    # Optional: Add security headers
    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "no-referrer-when-downgrade"
    }

    # Optional: Enable compression
    encode gzip

    # Optional: Add request logging
    log {
        output file /var/log/caddy/wallpaper-gacha.log
    }
}
```

Start Caddy:
```bash
caddy run --config /path/to/Caddyfile
```

## Systemd Service

Create a systemd service file at `/etc/systemd/system/wallpaper-gacha.service`:

```ini
[Unit]
Description=Wallpaper Gacha Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/wallpaper-gacha
ExecStart=/path/to/wallpaper-gacha/wallpaper-gacha
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Enable and start the service:
```bash
sudo systemctl enable wallpaper-gacha
sudo systemctl start wallpaper-gacha
sudo systemctl status wallpaper-gacha
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `server_port` | Port to listen on | 8080 |
| `server_host` | Host to bind to | localhost |
| `discord_client_id` | Discord OAuth Client ID | Required |
| `discord_client_secret` | Discord OAuth Client Secret | Required |
| `discord_redirect_uri` | OAuth callback URL | Required |
| `allowed_server_ids` | Array of Discord server IDs | Required |
| `upload_cooldown_minutes` | Minutes between uploads | 60 |
| `max_file_size_mb` | Maximum file size in MB | 50 |
| `database_path` | Path to SQLite database | ./wallpaper.db |
| `upload_directory` | Directory for uploaded files | ./uploads |
| `session_secret` | Secret key for sessions | Required |

## File Structure

```
wallpaper-gacha/
├── main.go                 # Application entry point
├── config/
│   └── config.go          # Configuration loader
├── handlers/
│   ├── auth.go            # Discord OAuth handlers
│   ├── upload.go          # Image upload handler
│   └── home.go            # Page handlers
├── middleware/
│   └── auth.go            # Authentication middleware
├── models/
│   ├── database.go        # Database initialization
│   └── user.go            # User and upload models
├── static/
│   ├── index.html         # Landing page
│   └── upload.html        # Upload page
├── uploads/               # Uploaded images (created automatically)
├── config.json            # Configuration file (you create this)
└── wallpaper.db          # SQLite database (created automatically)
```

## Database Schema

### Users Table
- `discord_id` (TEXT, PRIMARY KEY): User's Discord ID
- `username` (TEXT): User's Discord username
- `created_at` (DATETIME): When the user first logged in
- `last_upload_at` (DATETIME): Last upload timestamp

### Uploads Table
- `id` (INTEGER, PRIMARY KEY): Auto-incrementing ID
- `discord_id` (TEXT): Uploader's Discord ID
- `filename` (TEXT): Stored filename (UUID + extension)
- `original_filename` (TEXT): Original filename
- `file_size` (INTEGER): File size in bytes
- `uploaded_at` (DATETIME): Upload timestamp

## Security Features

- Session-based authentication with secure cookies
- Discord server membership verification
- File type validation (extension and MIME type)
- File size limits
- Rate limiting per user
- Unique filenames to prevent collisions
- SQLite with prepared statements (SQL injection protection)

## Troubleshooting

### "Failed to authenticate with Discord"
- Check that your Discord Client ID and Secret are correct
- Verify the redirect URI matches exactly in both config.json and Discord app settings

### "You are not in an allowed Discord server"
- Verify you copied the correct Server ID
- Make sure you're a member of the server
- Check that the server ID is in the `allowed_server_ids` array

### "Failed to save file"
- Check that the uploads directory exists and is writable
- Verify disk space is available
- Check file permissions

### Upload fails with no error
- Check browser console for JavaScript errors
- Verify the file size is under the limit
- Ensure the file format is supported

## Development

Run with live reload using Air:
```bash
go install github.com/cosmtrek/air@latest
air
```

Run tests:
```bash
go test ./...
```

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
