#!/usr/bin/env python3
"""
Wallpaper Gacha Manager - Local wallpaper rotation client
Downloads images from remote server and rotates them using hyprpaper
"""

import sqlite3
import subprocess
import time
import json
import os
import random
import logging
from pathlib import Path
from datetime import datetime
from typing import List, Optional


class WallpaperManager:
    def __init__(self, config_path: str = "wallpaper-manager-config.json"):
        """Initialize the wallpaper manager"""
        self.config = self.load_config(config_path)
        self.local_dir = Path(self.config["local_directory"]).expanduser()
        self.local_dir.mkdir(parents=True, exist_ok=True)

        self.db_path = self.local_dir / "wallpaper-history.db"
        self.init_database()

        # Setup logging
        log_level = getattr(logging, self.config.get("log_level", "INFO"))
        logging.basicConfig(
            level=log_level,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(self.local_dir / "wallpaper-manager.log"),
                logging.StreamHandler()
            ]
        )
        self.logger = logging.getLogger(__name__)

    def load_config(self, config_path: str) -> dict:
        """Load configuration from JSON file"""
        if not os.path.exists(config_path):
            raise FileNotFoundError(
                f"Configuration file not found: {config_path}\n"
                f"Please create it using wallpaper-manager-config.example.json as a template"
            )

        with open(config_path, 'r') as f:
            config = json.load(f)

        # Validate required fields
        required = ["remote_host", "remote_user", "remote_directory", "local_directory"]
        missing = [field for field in required if field not in config]
        if missing:
            raise ValueError(f"Missing required config fields: {missing}")

        # Set defaults
        config.setdefault("interval_minutes", 30)
        config.setdefault("rsync_port", 22)
        config.setdefault("log_level", "INFO")
        config.setdefault("hyprpaper_monitors", ["all"])

        return config

    def init_database(self):
        """Initialize SQLite database for tracking displayed images"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        cursor.execute("""
            CREATE TABLE IF NOT EXISTS wallpaper_history (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                filename TEXT NOT NULL UNIQUE,
                first_displayed_at DATETIME,
                last_displayed_at DATETIME,
                display_count INTEGER DEFAULT 0
            )
        """)

        cursor.execute("""
            CREATE TABLE IF NOT EXISTS rotation_state (
                id INTEGER PRIMARY KEY CHECK (id = 1),
                all_images_shown BOOLEAN DEFAULT 0,
                last_reset_at DATETIME
            )
        """)

        # Initialize rotation state if it doesn't exist
        cursor.execute("INSERT OR IGNORE INTO rotation_state (id) VALUES (1)")

        conn.commit()
        conn.close()

    def sync_images(self):
        """Download images from remote server using rsync"""
        remote_path = f"{self.config['remote_user']}@{self.config['remote_host']}:{self.config['remote_directory']}/"

        rsync_cmd = [
            "rsync",
            "-avz",
            "--include=*.png",
            "--include=*.jpg",
            "--include=*.jpeg",
            "--include=*.jxl",
            "--include=*.webp",
            "--exclude=*",
            "-e", f"ssh -p {self.config['rsync_port']}",
            remote_path,
            str(self.local_dir)
        ]

        self.logger.info(f"Syncing images from {remote_path}")

        try:
            result = subprocess.run(
                rsync_cmd,
                capture_output=True,
                text=True,
                check=True
            )

            if result.stdout:
                self.logger.debug(f"rsync output: {result.stdout}")

            self.logger.info("Image sync completed successfully")
            return True

        except subprocess.CalledProcessError as e:
            self.logger.error(f"Failed to sync images: {e.stderr}")
            return False
        except Exception as e:
            self.logger.error(f"Unexpected error during sync: {e}")
            return False

    def get_available_images(self) -> List[Path]:
        """Get list of all image files in local directory"""
        extensions = ['.png', '.jpg', '.jpeg', '.jxl', '.webp']
        images = []

        for ext in extensions:
            images.extend(self.local_dir.glob(f"*{ext}"))

        return images

    def get_unshown_images(self) -> List[Path]:
        """Get list of images that haven't been displayed yet in current rotation"""
        all_images = self.get_available_images()

        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Get state to check if we need to reset
        cursor.execute("SELECT all_images_shown FROM rotation_state WHERE id = 1")
        state = cursor.fetchone()
        all_shown = state[0] if state else False

        if all_shown:
            # Reset rotation - mark all images as unshown
            self.logger.info("All images have been shown. Starting new rotation cycle.")
            cursor.execute("UPDATE wallpaper_history SET display_count = 0")
            cursor.execute("UPDATE rotation_state SET all_images_shown = 0, last_reset_at = ?",
                         (datetime.now().isoformat(),))
            conn.commit()

        # Get filenames of images that have been shown (display_count > 0)
        cursor.execute("SELECT filename FROM wallpaper_history WHERE display_count > 0")
        shown_filenames = {row[0] for row in cursor.fetchall()}

        conn.close()

        # Filter out images that have been shown
        unshown = [img for img in all_images if img.name not in shown_filenames]

        self.logger.debug(f"Total images: {len(all_images)}, Unshown: {len(unshown)}")

        return unshown

    def mark_image_displayed(self, filename: str):
        """Mark an image as displayed in the database"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        now = datetime.now().isoformat()

        cursor.execute("""
            INSERT INTO wallpaper_history (filename, first_displayed_at, last_displayed_at, display_count)
            VALUES (?, ?, ?, 1)
            ON CONFLICT(filename) DO UPDATE SET
                last_displayed_at = ?,
                display_count = display_count + 1
        """, (filename, now, now, now))

        # Check if all images have been shown
        cursor.execute("SELECT COUNT(*) FROM wallpaper_history WHERE display_count > 0")
        shown_count = cursor.fetchone()[0]

        total_images = len(self.get_available_images())

        if shown_count >= total_images and total_images > 0:
            self.logger.info("All available images have been shown in this rotation")
            cursor.execute("UPDATE rotation_state SET all_images_shown = 1 WHERE id = 1")

        conn.commit()
        conn.close()

    def set_wallpaper_hyprpaper(self, image_path: Path) -> bool:
        """Set wallpaper using hyprpaper"""
        try:
            # First, preload the image
            preload_cmd = ["hyprctl", "hyprpaper", "preload", str(image_path)]
            result = subprocess.run(preload_cmd, capture_output=True, text=True)

            if result.returncode != 0:
                self.logger.error(f"Failed to preload image: {result.stderr}")
                return False

            # Then set it as wallpaper for each monitor
            monitors = self.config.get("hyprpaper_monitors", ["all"])

            if "all" in monitors:
                # Get all monitors
                monitors_cmd = ["hyprctl", "monitors", "-j"]
                result = subprocess.run(monitors_cmd, capture_output=True, text=True)

                if result.returncode == 0:
                    try:
                        monitors_data = json.loads(result.stdout)
                        monitors = [m["name"] for m in monitors_data]
                    except json.JSONDecodeError:
                        self.logger.warning("Failed to parse monitors, using default")
                        monitors = ["DP-1"]  # Fallback

            # Set wallpaper for each monitor
            for monitor in monitors:
                wallpaper_cmd = ["hyprctl", "hyprpaper", "wallpaper", f"{monitor},{image_path}"]
                result = subprocess.run(wallpaper_cmd, capture_output=True, text=True)

                if result.returncode != 0:
                    self.logger.error(f"Failed to set wallpaper on {monitor}: {result.stderr}")
                else:
                    self.logger.debug(f"Set wallpaper on {monitor}")

            self.logger.info(f"Successfully set wallpaper: {image_path.name}")
            return True

        except Exception as e:
            self.logger.error(f"Failed to set wallpaper: {e}")
            return False

    def change_wallpaper(self):
        """Select and display a new wallpaper"""
        unshown_images = self.get_unshown_images()

        if not unshown_images:
            self.logger.warning("No unshown images available")
            # Try syncing to get new images
            if self.sync_images():
                unshown_images = self.get_unshown_images()

            if not unshown_images:
                self.logger.error("No images available even after sync")
                return False

        # Select random image from unshown ones
        selected_image = random.choice(unshown_images)

        self.logger.info(f"Selected wallpaper: {selected_image.name}")

        # Set the wallpaper
        if self.set_wallpaper_hyprpaper(selected_image):
            self.mark_image_displayed(selected_image.name)
            return True

        return False

    def run_once(self):
        """Run one cycle: sync and change wallpaper"""
        self.logger.info("Running wallpaper rotation cycle")

        # Sync images from server
        self.sync_images()

        # Change wallpaper
        self.change_wallpaper()

    def run_daemon(self):
        """Run continuously, changing wallpaper at intervals"""
        self.logger.info(f"Starting wallpaper manager daemon (interval: {self.config['interval_minutes']} minutes)")

        # Do initial run
        self.run_once()

        # Then run at intervals
        interval_seconds = self.config['interval_minutes'] * 60

        try:
            while True:
                time.sleep(interval_seconds)
                self.run_once()

        except KeyboardInterrupt:
            self.logger.info("Wallpaper manager stopped by user")
        except Exception as e:
            self.logger.error(f"Unexpected error in daemon loop: {e}")
            raise


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Wallpaper Gacha Manager")
    parser.add_argument(
        "--config",
        default="wallpaper-manager-config.json",
        help="Path to configuration file"
    )
    parser.add_argument(
        "--once",
        action="store_true",
        help="Run once and exit (useful for testing)"
    )

    args = parser.parse_args()

    try:
        manager = WallpaperManager(args.config)

        if args.once:
            manager.run_once()
        else:
            manager.run_daemon()

    except FileNotFoundError as e:
        print(f"Error: {e}")
        return 1
    except ValueError as e:
        print(f"Configuration error: {e}")
        return 1
    except Exception as e:
        print(f"Unexpected error: {e}")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())
