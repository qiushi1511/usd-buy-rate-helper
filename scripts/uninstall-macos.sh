#!/bin/bash

# Uninstallation script for ratemon daemon on macOS
set -e

PLIST_FILE="com.github.qiushi1511.ratemon.plist"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

echo "====================================="
echo "Ratemon Daemon Uninstallation (macOS)"
echo "====================================="
echo ""

# Unload the LaunchAgent
if [ -f "$LAUNCH_AGENTS_DIR/$PLIST_FILE" ]; then
    echo "Stopping daemon..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_FILE" 2>/dev/null || true

    echo "Removing LaunchAgent..."
    rm "$LAUNCH_AGENTS_DIR/$PLIST_FILE"

    echo ""
    echo "✅ Uninstallation complete!"
    echo ""
    echo "The daemon has been stopped and removed from auto-start."
    echo "Your data in ./data/ and logs in ./logs/ have been preserved."
else
    echo "❌ LaunchAgent not found. The daemon may not be installed."
fi
