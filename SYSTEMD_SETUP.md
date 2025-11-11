# Systemd Service Setup Guide

This guide will help you set up wallpaper-gacha to run automatically on system startup using systemd.

## Prerequisites

- Caddy is already installed and configured as a systemd service
- The wallpaper-gacha binary is built and located at `~/wallpaper-gacha/bin/wallpaper-gacha`
- Configuration file exists at `~/wallpaper-gacha/config.json` (the service will run from the bin directory and reference the config in the parent directory)

## Installation Steps

### 1. Customize the Service File

The included `wallpaper-gacha.service` file uses placeholders that need to be replaced:

```bash
# Replace %USER% with your username and %HOME% with your home directory
sed -e "s/%USER%/$USER/g" -e "s|%HOME%|$HOME|g" wallpaper-gacha.service > wallpaper-gacha-ready.service
```

### 2. Install the Service File

Copy the customized service file to the systemd directory:

```bash
sudo cp wallpaper-gacha-ready.service /etc/systemd/system/wallpaper-gacha.service
```

Or manually edit and copy:

```bash
# Edit the file to replace %USER% with your username and %HOME% with your home directory
sudo nano wallpaper-gacha.service

# Then copy it
sudo cp wallpaper-gacha.service /etc/systemd/system/
```

### 3. Reload Systemd

Tell systemd to reload its configuration:

```bash
sudo systemctl daemon-reload
```

### 4. Enable the Service

Enable the service to start on boot:

```bash
sudo systemctl enable wallpaper-gacha
```

### 5. Start the Service

Start the service immediately:

```bash
sudo systemctl start wallpaper-gacha
```

### 6. Verify the Service is Running

Check the status:

```bash
sudo systemctl status wallpaper-gacha
```

You should see output indicating the service is "active (running)".

## Managing the Service

### View Logs

View real-time logs:

```bash
sudo journalctl -u wallpaper-gacha -f
```

View recent logs:

```bash
sudo journalctl -u wallpaper-gacha -n 50
```

View logs since last boot:

```bash
sudo journalctl -u wallpaper-gacha -b
```

### Restart the Service

After making configuration changes:

```bash
sudo systemctl restart wallpaper-gacha
```

### Stop the Service

```bash
sudo systemctl stop wallpaper-gacha
```

### Disable Auto-start

```bash
sudo systemctl disable wallpaper-gacha
```

## Service Features

### Automatic Restart

The service is configured to automatically restart if it crashes:
- `Restart=on-failure`: Restarts only if the process exits with an error
- `RestartSec=5s`: Waits 5 seconds between restart attempts

### Dependency Management

- Starts after Caddy is running (`After=caddy.service`)
- Requires Caddy to be running (`Requires=caddy.service`)
- If Caddy stops, this service will also stop

### Security Hardening

The service includes several security features:
- `NoNewPrivileges=true`: Prevents privilege escalation
- `PrivateTmp=true`: Uses a private /tmp directory
- `ProtectSystem=strict`: Makes most of the filesystem read-only
- `ProtectHome=read-only`: Makes home directories read-only except for specified paths
- `ReadWritePaths`: Only allows writing to uploads directory and database

### Logging

All output is sent to the systemd journal and can be viewed with `journalctl`.

## Troubleshooting

### Service won't start

1. Check the service status:
   ```bash
   sudo systemctl status wallpaper-gacha
   ```

2. Check the logs for error messages:
   ```bash
   sudo journalctl -u wallpaper-gacha -n 100
   ```

3. Common issues:
   - **Binary not found**: Ensure the binary exists at `~/wallpaper-gacha/bin/wallpaper-gacha`
   - **Config file missing**: Ensure `config.json` exists in `~/wallpaper-gacha/`
   - **Permission errors**: Check that the user has read/write access to the working directory
   - **Caddy not running**: Start Caddy first: `sudo systemctl start caddy`

### Permission Denied Errors

If you see permission errors:

1. Check file ownership:
   ```bash
   ls -la ~/wallpaper-gacha/
   ```

2. Ensure your user owns the files:
   ```bash
   chown -R $USER:$USER ~/wallpaper-gacha/
   ```

3. Ensure the binary is executable:
   ```bash
   chmod +x ~/wallpaper-gacha/bin/wallpaper-gacha
   ```

### Database Lock Errors

If you see "database is locked" errors:
- Ensure no other instance of wallpaper-gacha is running
- Check that the database file has correct permissions

## Advanced Configuration

### Custom Binary Location

If your binary is in a different location, edit the service file:

```ini
ExecStart=/custom/path/to/wallpaper-gacha
```

### Custom Config File

The service is configured to load config from the parent directory (`../config.json`). To use a different config file location:

```ini
ExecStart=%HOME%/wallpaper-gacha/bin/wallpaper-gacha /path/to/custom-config.json
```

Or for a relative path from the bin directory:

```ini
ExecStart=%HOME%/wallpaper-gacha/bin/wallpaper-gacha ../custom-config.json
```

### Environment Variables

To add environment variables:

```ini
[Service]
Environment="VARIABLE_NAME=value"
Environment="ANOTHER_VAR=another_value"
```

### Different User

To run as a different user (not recommended unless necessary):

```ini
[Service]
User=different-user
Group=different-group
```

Make sure that user has access to the files and directories.

## Uninstalling

To completely remove the service:

```bash
# Stop the service
sudo systemctl stop wallpaper-gacha

# Disable it
sudo systemctl disable wallpaper-gacha

# Remove the service file
sudo rm /etc/systemd/system/wallpaper-gacha.service

# Reload systemd
sudo systemctl daemon-reload
```
