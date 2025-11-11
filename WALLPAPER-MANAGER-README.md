# Wallpaper Gacha Manager - Local Client

A Python-based wallpaper rotation manager that downloads images from your Wallpaper Gacha server and automatically changes your wallpaper using hyprpaper on Hyprland.

## Features

- Downloads images from remote server using rsync over SSH
- Automatic wallpaper rotation at configurable intervals
- Smart image tracking - shows each image once before repeating
- Seamless integration with hyprpaper
- SQLite-based history tracking
- Configurable logging
- Systemd service for automatic startup

## Prerequisites

- Python 3.6 or higher
- Hyprland with hyprpaper
- SSH access to your Wallpaper Gacha server
- rsync installed on both local and remote machines

## Installation

### 1. Set up SSH key authentication

For passwordless operation, set up SSH key authentication to your server:

```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -C "wallpaper-gacha"

# Copy your public key to the server
ssh-copy-id username@your-server.com
```

### 2. Create configuration file

```bash
cd /path/to/wallpaper-gacha
cp wallpaper-manager-config.example.json wallpaper-manager-config.json
```

Edit `wallpaper-manager-config.json` with your settings:

```json
{
  "remote_host": "your-server.com",
  "remote_user": "your-username",
  "remote_directory": "/path/to/wallpaper-gacha/uploads",
  "remote_port": 22,
  "local_directory": "~/.local/share/wallpaper-gacha",
  "interval_minutes": 30,
  "rsync_port": 22,
  "hyprpaper_monitors": ["all"],
  "log_level": "INFO"
}
```

#### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `remote_host` | Hostname or IP of your server | Required |
| `remote_user` | SSH username | Required |
| `remote_directory` | Path to uploads folder on server | Required |
| `remote_port` | SSH port (same as rsync_port) | 22 |
| `local_directory` | Local directory to store images | Required |
| `interval_minutes` | Minutes between wallpaper changes | 30 |
| `rsync_port` | SSH/rsync port | 22 |
| `hyprpaper_monitors` | Monitor names or ["all"] | ["all"] |
| `log_level` | Logging level (DEBUG, INFO, WARNING, ERROR) | INFO |

### 3. Make the script executable

```bash
chmod +x wallpaper-manager.py
```

### 4. Test the script

Run once to test:

```bash
./wallpaper-manager.py --once
```

This will:
1. Download images from your server
2. Select a random image
3. Set it as your wallpaper
4. Exit

Check the logs at `~/.local/share/wallpaper-gacha/wallpaper-manager.log`

## Running as a Service

To have the wallpaper manager start automatically with your session:

### 1. Install the systemd user service

```bash
# Create systemd user directory if it doesn't exist
mkdir -p ~/.config/systemd/user

# Copy the service file
cp wallpaper-manager.service ~/.config/systemd/user/

# Edit the service file if your paths are different
nano ~/.config/systemd/user/wallpaper-manager.service
```

**Important:** Update the `WorkingDirectory` and `ExecStart` paths in the service file to match your installation location.

### 2. Enable and start the service

```bash
# Reload systemd
systemctl --user daemon-reload

# Enable the service to start automatically
systemctl --user enable wallpaper-manager.service

# Start the service now
systemctl --user start wallpaper-manager.service

# Check status
systemctl --user status wallpaper-manager.service
```

### 3. View logs

```bash
# Follow live logs
journalctl --user -u wallpaper-manager.service -f

# View recent logs
journalctl --user -u wallpaper-manager.service -n 50
```

## Manual Usage

### Run in daemon mode (continuous)

```bash
./wallpaper-manager.py
```

The script will:
1. Sync images from the server
2. Change your wallpaper
3. Wait for the configured interval
4. Repeat

### Run once (for testing)

```bash
./wallpaper-manager.py --once
```

### Specify custom config file

```bash
./wallpaper-manager.py --config /path/to/config.json
```

## How It Works

### Image Tracking

The manager uses a local SQLite database (`wallpaper-history.db`) to track which images have been displayed:

1. **First Rotation:** Shows all images once in random order
2. **Completion:** When all images have been shown, it automatically resets
3. **New Images:** When new images are downloaded, they're added to the pool
4. **No Repeats:** Images won't repeat until all others have been shown

### Wallpaper Changes

The manager integrates with hyprpaper:

1. Downloads images via rsync
2. Selects a random unshown image
3. Uses `hyprctl hyprpaper preload` to load the image
4. Uses `hyprctl hyprpaper wallpaper` to set it on your monitor(s)
5. Marks the image as displayed in the database

### Monitor Configuration

By default, the manager sets the wallpaper on all monitors. To target specific monitors:

1. Get your monitor names:
   ```bash
   hyprctl monitors
   ```

2. Update config:
   ```json
   {
     "hyprpaper_monitors": ["DP-1", "HDMI-A-1"]
   }
   ```

## Troubleshooting

### "Failed to sync images"

Check:
- SSH key authentication is working: `ssh username@your-server.com`
- Remote directory path is correct
- Remote directory is readable
- rsync is installed on both machines

### "Failed to set wallpaper"

Check:
- Hyprland is running
- hyprpaper is running (`pgrep hyprpaper`)
- Monitor names are correct (`hyprctl monitors`)

### Service not starting

Check:
- Service file paths are correct
- Python script is executable
- Config file exists and is valid JSON
- View logs: `journalctl --user -u wallpaper-manager.service -n 50`

### Images not changing

Check:
- Service is running: `systemctl --user status wallpaper-manager.service`
- Logs for errors: `~/.local/share/wallpaper-gacha/wallpaper-manager.log`
- Images were downloaded: `ls ~/.local/share/wallpaper-gacha/`

### Permission denied errors

Make sure:
- Local directory is writable
- SSH key has correct permissions (600 for private key)
- User has access to the remote directory

## Advanced Configuration

### Custom Interval

Change wallpaper every 5 minutes:
```json
{
  "interval_minutes": 5
}
```

### Debug Logging

For troubleshooting:
```json
{
  "log_level": "DEBUG"
}
```

### Specific Remote Port

If SSH is on a non-standard port:
```json
{
  "remote_port": 2222,
  "rsync_port": 2222
}
```

## File Structure

```
~/.local/share/wallpaper-gacha/
├── *.png, *.jpg, etc.        # Downloaded wallpaper images
├── wallpaper-history.db      # Image tracking database
└── wallpaper-manager.log     # Application logs
```

## Stopping the Service

```bash
# Stop the service
systemctl --user stop wallpaper-manager.service

# Disable auto-start
systemctl --user disable wallpaper-manager.service
```

## Uninstallation

```bash
# Stop and disable service
systemctl --user stop wallpaper-manager.service
systemctl --user disable wallpaper-manager.service

# Remove service file
rm ~/.config/systemd/user/wallpaper-manager.service

# Remove downloaded images and database
rm -rf ~/.local/share/wallpaper-gacha

# Remove config (if desired)
rm wallpaper-manager-config.json
```

## Tips

1. **Testing:** Always test with `--once` flag before running as a service
2. **SSH Keys:** Use a dedicated SSH key for security
3. **Bandwidth:** The sync only downloads new/changed images
4. **Privacy:** All tracking is local - nothing is sent back to the server
5. **Monitoring:** Check logs regularly for any sync issues

## License

See LICENSE file for details.
