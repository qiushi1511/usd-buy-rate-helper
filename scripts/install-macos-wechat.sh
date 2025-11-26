#!/bin/bash

# Installation script for ratemon daemon on macOS with WeChat notifications
set -e

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PLIST_FILE="com.github.qiushi1511.ratemon.plist"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

echo "==================================================="
echo "Ratemon Daemon Installation (macOS + WeChat Alerts)"
echo "==================================================="
echo ""

# Check if WeChat webhook URL is provided
if [ -z "$1" ]; then
    echo "❌ Error: WeChat webhook URL is required"
    echo ""
    echo "Usage: $0 <wechat-webhook-url> [alert-high] [alert-low] [alert-change]"
    echo ""
    echo "Example:"
    echo "  $0 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY' 7.15 6.95 0.5"
    echo ""
    echo "Parameters:"
    echo "  wechat-webhook-url  - Required: Your WeChat Work group robot webhook URL"
    echo "  alert-high          - Optional: Alert when rate exceeds this (e.g., 7.15)"
    echo "  alert-low           - Optional: Alert when rate drops below this (e.g., 6.95)"
    echo "  alert-change        - Optional: Alert on % change (e.g., 0.5 for 0.5%)"
    echo ""
    exit 1
fi

WECHAT_WEBHOOK="$1"
ALERT_HIGH="${2:-}"
ALERT_LOW="${3:-}"
ALERT_CHANGE="${4:-}"

echo "Configuration:"
echo "  Project dir:     $PROJECT_DIR"
echo "  WeChat webhook:  ${WECHAT_WEBHOOK:0:50}..."
[ -n "$ALERT_HIGH" ] && echo "  Alert high:      $ALERT_HIGH CNY"
[ -n "$ALERT_LOW" ] && echo "  Alert low:       $ALERT_LOW CNY"
[ -n "$ALERT_CHANGE" ] && echo "  Alert change:    $ALERT_CHANGE%"
echo ""

# Create logs directory
echo "Creating logs directory..."
mkdir -p "$PROJECT_DIR/logs"

# Create data directory (if not exists)
echo "Creating data directory..."
mkdir -p "$PROJECT_DIR/data"

# Generate plist file from template
echo "Generating LaunchAgent plist..."
cat > "/tmp/$PLIST_FILE" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.qiushi1511.ratemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>$PROJECT_DIR/ratemon</string>
        <string>daemon</string>
        <string>--db</string>
        <string>$PROJECT_DIR/data/rates.db</string>
        <string>--migrations</string>
        <string>$PROJECT_DIR/migrations</string>
        <string>--wechat-webhook</string>
        <string>$WECHAT_WEBHOOK</string>
EOF

# Add alert thresholds if provided
if [ -n "$ALERT_HIGH" ]; then
    cat >> "/tmp/$PLIST_FILE" <<EOF
        <string>--alert-high</string>
        <string>$ALERT_HIGH</string>
EOF
fi

if [ -n "$ALERT_LOW" ]; then
    cat >> "/tmp/$PLIST_FILE" <<EOF
        <string>--alert-low</string>
        <string>$ALERT_LOW</string>
EOF
fi

if [ -n "$ALERT_CHANGE" ]; then
    cat >> "/tmp/$PLIST_FILE" <<EOF
        <string>--alert-change</string>
        <string>$ALERT_CHANGE</string>
EOF
fi

# Complete the plist file
cat >> "/tmp/$PLIST_FILE" <<EOF
    </array>

    <key>WorkingDirectory</key>
    <string>$PROJECT_DIR</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>$PROJECT_DIR/logs/ratemon.log</string>

    <key>StandardErrorPath</key>
    <string>$PROJECT_DIR/logs/ratemon.error.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOF

# Stop existing daemon if running
if launchctl list | grep -q "com.github.qiushi1511.ratemon"; then
    echo "Stopping existing daemon..."
    launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_FILE" 2>/dev/null || true
fi

# Install the plist
echo "Installing LaunchAgent..."
mkdir -p "$LAUNCH_AGENTS_DIR"
cp "/tmp/$PLIST_FILE" "$LAUNCH_AGENTS_DIR/$PLIST_FILE"

# Load the LaunchAgent
echo "Loading LaunchAgent..."
launchctl load "$LAUNCH_AGENTS_DIR/$PLIST_FILE"

# Wait a moment and check status
sleep 2
if launchctl list | grep -q "com.github.qiushi1511.ratemon"; then
    echo ""
    echo "✅ Installation complete!"
    echo ""
    echo "The ratemon daemon is now running with WeChat notifications enabled!"
    echo ""
    echo "📱 Alerts will be sent to your WeChat Work group when:"
    [ -n "$ALERT_HIGH" ] && echo "   • Rate exceeds $ALERT_HIGH CNY"
    [ -n "$ALERT_LOW" ] && echo "   • Rate drops below $ALERT_LOW CNY"
    [ -n "$ALERT_CHANGE" ] && echo "   • Rate changes by $ALERT_CHANGE% or more"
    echo ""
    echo "The daemon will:"
    echo "  • Start automatically when you log in"
    echo "  • Restart automatically if it crashes"
    echo "  • Poll the CMB API every minute (during business hours 08:00-22:00 CST)"
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
    echo "  $PROJECT_DIR/ratemon monitor --once"
    echo "  $PROJECT_DIR/ratemon history --last 1h"
    echo "  $PROJECT_DIR/ratemon patterns --days 7"
    echo ""
else
    echo ""
    echo "⚠️  Warning: Daemon may not have started successfully."
    echo "Check logs for errors:"
    echo "  tail $PROJECT_DIR/logs/ratemon.error.log"
    echo ""
fi
