# Daemon Setup Guide (macOS)

This guide shows how to run the ratemon daemon as a background service on macOS using launchd.

## Quick Start

### Install and Start Daemon

```bash
# Run the installation script
./scripts/install-macos.sh
```

This will:
- ✅ Create necessary directories (logs, data)
- ✅ Install the LaunchAgent configuration
- ✅ Start the daemon immediately
- ✅ Configure auto-start on login

### Verify It's Running

```bash
# Check if daemon is running
launchctl list | grep ratemon

# Should show something like:
# 12345   0   com.github.qiushi1511.ratemon

# View real-time logs
tail -f logs/ratemon.log

# Check collected data
./ratemon monitor --once
```

## Managing the Daemon

### View Logs

```bash
# Standard output (info logs)
tail -f logs/ratemon.log

# Error output
tail -f logs/ratemon.error.log

# Last 50 lines
tail -50 logs/ratemon.log
```

### Stop the Daemon

```bash
launchctl unload ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist
```

### Start the Daemon

```bash
launchctl load ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist
```

### Restart the Daemon

```bash
# Method 1: Unload and reload
launchctl unload ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist
launchctl load ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist

# Method 2: Kickstart (faster)
launchctl kickstart -k gui/$(id -u)/com.github.qiushi1511.ratemon
```

### Check Status

```bash
# List all user LaunchAgents
launchctl list | grep ratemon

# View detailed status
launchctl print gui/$(id -u)/com.github.qiushi1511.ratemon
```

## Uninstall

To completely remove the daemon:

```bash
./scripts/uninstall-macos.sh
```

This will:
- Stop the daemon
- Remove from auto-start
- Keep your data and logs intact

## Troubleshooting

### Daemon Not Starting

1. **Check logs for errors:**
   ```bash
   cat logs/ratemon.error.log
   ```

2. **Verify binary exists:**
   ```bash
   ls -l ratemon
   ```

3. **Test manually:**
   ```bash
   ./ratemon daemon -v
   ```

### Permission Issues

```bash
# Make sure directories are writable
chmod 755 data logs

# Verify binary is executable
chmod +x ratemon
```

### Rebuild After Code Changes

If you modify the code:

```bash
# Rebuild binary
go build -o ratemon ./cmd/ratemon

# Restart daemon to use new binary
launchctl kickstart -k gui/$(id -u)/com.github.qiushi1511.ratemon
```

### View System Logs

```bash
# Check macOS system logs for launchd issues
log show --predicate 'process == "launchd"' --last 1h | grep ratemon
```

## Configuration

The LaunchAgent configuration is located at:
```
~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist
```

Key settings:
- **RunAtLoad**: Start on login
- **KeepAlive**: Restart if crashes
- **WorkingDirectory**: Project directory
- **StandardOutPath**: Where logs are written
- **StandardErrorPath**: Where errors are written

To modify settings:
1. Edit `scripts/com.github.qiushi1511.ratemon.plist`
2. Run `./scripts/install-macos.sh` again

## Alternative: Manual Background Process

If you prefer not to use launchd, you can run the daemon manually:

```bash
# Run in background with nohup
nohup ./ratemon daemon > logs/ratemon.log 2>&1 &

# Get process ID
echo $! > ratemon.pid

# To stop:
kill $(cat ratemon.pid)
```

## Data Location

- **Database**: `./data/rates.db`
- **Logs**: `./logs/ratemon.log` and `./logs/ratemon.error.log`
- **Migrations**: `./migrations/`

## Auto-Start on Boot

The LaunchAgent is configured to start when you log in. To disable auto-start but keep the daemon installed:

```bash
# Unload (stop and disable auto-start)
launchctl unload ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist

# Load manually when needed
launchctl load ~/Library/LaunchAgents/com.github.qiushi1511.ratemon.plist
```
