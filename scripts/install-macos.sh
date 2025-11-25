#!/bin/bash

# Installation script for ratemon daemon on macOS
set -e

PROJECT_DIR="/Users/junhuif/Codes/Go/src/github.com/qiushi1511/usd-buy-rate-monitor"
PLIST_FILE="com.github.qiushi1511.ratemon.plist"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

echo "==================================="
echo "Ratemon Daemon Installation (macOS)"
echo "==================================="
echo ""

# Create logs directory
echo "Creating logs directory..."
mkdir -p "$PROJECT_DIR/logs"

# Create data directory (if not exists)
echo "Creating data directory..."
mkdir -p "$PROJECT_DIR/data"

# Copy plist to LaunchAgents directory
echo "Installing LaunchAgent..."
mkdir -p "$LAUNCH_AGENTS_DIR"
cp "$PROJECT_DIR/scripts/$PLIST_FILE" "$LAUNCH_AGENTS_DIR/$PLIST_FILE"

# Load the LaunchAgent
echo "Loading LaunchAgent..."
launchctl load "$LAUNCH_AGENTS_DIR/$PLIST_FILE"

echo ""
echo "✅ Installation complete!"
echo ""
echo "The ratemon daemon is now running and will:"
echo "  - Start automatically when you log in"
echo "  - Restart automatically if it crashes"
echo "  - Poll the CMB API every minute"
echo ""
echo "Useful commands:"
echo "  Check status:    launchctl list | grep ratemon"
echo "  View logs:       tail -f $PROJECT_DIR/logs/ratemon.log"
echo "  View errors:     tail -f $PROJECT_DIR/logs/ratemon.error.log"
echo "  Stop daemon:     launchctl unload $LAUNCH_AGENTS_DIR/$PLIST_FILE"
echo "  Start daemon:    launchctl load $LAUNCH_AGENTS_DIR/$PLIST_FILE"
echo "  Restart daemon:  launchctl kickstart -k gui/\$(id -u)/com.github.qiushi1511.ratemon"
echo ""
echo "Query collected data:"
echo "  ./ratemon monitor --once"
echo "  ./ratemon history --last 1h"
echo ""
