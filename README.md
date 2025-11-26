# USD/CNY Exchange Rate Monitor

A lightweight monitoring system that tracks the USD/CNY exchange rate from China Merchants Bank's API and provides historical analysis.

## Features

- **Background Polling**: Continuously polls CMB API every minute for USD exchange rates
- **Smart Business Hours**: Automatically skips polling outside CMB business hours (08:00-22:00 CST) to save resources
- **SQLite Storage**: Stores historical data locally with date-based partitioning
- **Real-time Monitoring**: Display current exchange rate with live updates
- **Historical Analysis**: Query rates by time range with multiple output formats
- **Daily Statistics**: Peak rate analysis and average calculations
- **Pattern Recognition**: Discover hourly and weekly patterns to predict optimal exchange times
- **Alert System**: Get notified when rates cross thresholds or show unusual patterns
- **WeChat Work Integration**: Send alerts to WeChat Work (企业微信) group chats in Chinese
- **Data Retention**: Intelligent multi-tier aggregation (99.7% storage reduction while maintaining prediction accuracy)
- **ASCII Charts**: Visualize exchange rate trends directly in the terminal
- **Graceful Shutdown**: Handles Ctrl+C and SIGTERM signals properly

## Prerequisites

- Go 1.21 or later
- SQLite3 (included via go-sqlite3 driver)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/qiushi1511/usd-buy-rate-monitor.git
cd usd-buy-rate-monitor
```

2. Install dependencies:

```bash
go mod download
```

3. Build the binary:

```bash
go build -o ratemon ./cmd/ratemon
```

## Usage

### Run the Daemon

Start the background polling service that collects USD exchange rates every minute:

```bash
./ratemon daemon
```

**Options:**

- `-i, --interval duration` - Polling interval (default: 1m)
- `--no-business-hours` - Disable business hours check (poll 24/7)
- `--alert-high float` - Alert when rate exceeds this threshold
- `--alert-low float` - Alert when rate drops below this threshold
- `--alert-change float` - Alert when rate changes by this percent (e.g., 0.5 for 0.5%)
- `--alert-pattern` - Alert on unusual patterns (deviation from historical)
- `--alert-pattern-stddev float` - Std deviations for pattern alerts (default: 2.0)
- `--alert-cooldown int` - Minutes between repeat alerts of same type (default: 60)
- `-d, --db string` - Database file path (default: ./data/rates.db)
- `-m, --migrations string` - Migrations directory path (default: ./migrations)
- `-v, --verbose` - Enable verbose logging

**Business Hours:**
By default, the daemon only polls the CMB API during business hours (08:00-22:00 CST) since exchange rates don't update outside these hours. This reduces unnecessary API calls by ~60%. Use `--no-business-hours` to disable this optimization and poll 24/7.

**Alert System:**
The daemon can monitor rates and send alerts when certain conditions are met:

- **Threshold Alerts**: Notify when rate goes above/below specified values
- **Change Alerts**: Notify when rate changes significantly in short time
- **Pattern Alerts**: Notify when current rate deviates from historical patterns

Alerts are logged to stdout/stderr and can be sent to **WeChat Work (企业微信)** group chats in Chinese.

**Examples:**

```bash
# Standard daemon with business hours optimization
./ratemon daemon

# Poll every 30 seconds with verbose logging
./ratemon daemon -i 30s -v

# Alert if rate exceeds 7.10 or drops below 7.00
./ratemon daemon --alert-high 7.10 --alert-low 7.00

# Alert on 0.5% changes within polling interval
./ratemon daemon --alert-change 0.5

# Alert on unusual patterns (2 std deviations from historical average)
./ratemon daemon --alert-pattern

# Combine multiple alert types
./ratemon daemon --alert-high 7.15 --alert-low 6.95 --alert-change 0.3 --alert-pattern

# Send alerts to WeChat Work group chat
./ratemon daemon --alert-high 7.10 --wechat-webhook 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY'

# Full configuration with WeChat notifications
./ratemon daemon \
  --alert-high 7.15 \
  --alert-low 6.95 \
  --alert-change 0.5 \
  --alert-pattern \
  --wechat-webhook 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY'

# Disable business hours check (poll 24/7)
./ratemon daemon --no-business-hours

# Use a custom database path
./ratemon daemon -d /var/lib/ratemon/rates.db
```

### Monitor Current Rate

Display the current/latest exchange rate:

```bash
# Show current rate once
./ratemon monitor --once

# Live monitoring with auto-refresh every 10 seconds (default)
./ratemon monitor

# Custom refresh interval
./ratemon monitor --refresh 5s
```

**Example Output:**

```
USD/CNY Exchange Rate
═════════════════════

  Rate:      7.0749 CNY
  Time:      2025-11-25 20:25:30
  Age:       15s ago

  Change:    ↑ 0.0012 (0.02%)
  Previous:  7.0737 CNY at 20:24:30
```

### Query Historical Data

View exchange rates for specific time ranges:

```bash
# Last 2 hours
./ratemon history --last 2h

# Specific time range (today)
./ratemon history --start "09:00" --end "17:00"

# Specific date and time
./ratemon history --start "2025-11-25 09:00" --end "2025-11-25 17:00"

# Output as CSV
./ratemon history --last 1h --format csv

# Output as JSON
./ratemon history --last 30m --format json

# Show ASCII chart with data
./ratemon history --last 2h --chart

# Only chart, no table
./ratemon history --last 2h --format chart
```

**Example Output (table format):**

```
Exchange Rate History
═════════════════════
Period: 2025-11-25 19:00:00 to 2025-11-25 21:00:00
Records: 120

Summary Statistics:
  Min:     7.0712 CNY
  Max:     7.0789 CNY
  Average: 7.0751 CNY
  Range:   0.0077 CNY

Time                  Rate (CNY)  Change
──────────────────────────────────────────────────
2025-11-25 19:01:00      7.0745     -
2025-11-25 19:02:00      7.0748   ↑+0.0003
2025-11-25 19:03:00      7.0751   ↑+0.0003
...
```

**Example Output (with chart):**

```
 7.08 ┼╮
 7.08 ┤╰╮
 7.08 ┤ ╰╮
 7.08 ┤  ╰─╮
 7.08 ┤    ╰╮
 7.07 ┤     ╰─╮
 7.07 ┤       ╰╮
 7.07 ┤        ╰─╮
 7.07 ┤          ╰╮
 7.07 ┤           ╰──╮
 7.07 ┤              ╰───
                    USD/CNY Rate (19:00 to 21:00)

Statistics: Min=7.0712  Max=7.0789  Avg=7.0751  Range=0.0077  Samples=120
```

### Daily Peak Analysis

Show the highest exchange rate for each day:

```bash
# Last 7 days (default)
./ratemon peak

# Last 30 days
./ratemon peak --days 30

# Specific dates
./ratemon peak 2025-11-25 2025-11-24 2025-11-23
```

**Example Output:**

```
Daily Peak Exchange Rates (Last 7 Days)
═════════════════════════════════════════

Date          Peak (CNY)  Time
────────────────────────────────────────
2025-11-25      7.0789  14:23:15
2025-11-24      7.0812  16:45:30
2025-11-23      7.0798  11:20:45

Summary:
  Highest Peak:   7.0812 CNY
  Lowest Peak:    7.0789 CNY
  Average Peak:   7.0800 CNY
  Peak Range:     0.0023 CNY
```

### Daily Average Analysis

Calculate average exchange rates for each day:

```bash
# Last 7 days (default)
./ratemon average

# Last 30 days with comparison
./ratemon average --days 30 --compare

# Specific dates with comparison
./ratemon average 2025-11-25 2025-11-24 --compare

# Show ASCII charts for trends
./ratemon average --days 7 --compare --chart
```

**Example Output:**

```
Daily Average Exchange Rates (Last 7 Days)
═══════════════════════════════════════════

Date          Average    Min        Max        Volatility  Samples
───────────────────────────────────────────────────────────────────
2025-11-25    7.0751    7.0712    7.0789    0.0077       1440
2025-11-24    7.0785    7.0745    7.0812    0.0067       1440
2025-11-23    7.0772    7.0730    7.0798    0.0068       1440

Comparison Across Dates
══════════════════════

  Overall Average:    7.0770 CNY
  Absolute Minimum:   7.0712 CNY
  Absolute Maximum:   7.0812 CNY
  Total Range:        0.0100 CNY

Day-to-Day Changes:
  2025-11-24 → 2025-11-25:  ↓ -0.0034 (-0.04%) [down]
  2025-11-23 → 2025-11-24:  ↑ 0.0013 (0.02%) [up]

Volatility Analysis:
  Average Daily Range:  0.0071 CNY
  Most Volatile Day:    2025-11-25 (0.0077 CNY range)
  Least Volatile Day:   2025-11-24 (0.0067 CNY range)
```

**Example Output (with charts):**

```
 7.08 ┼╮
 7.08 ┤╰─╮
 7.08 ┤  ╰╮
 7.08 ┤   ╰─╮
 7.08 ┤     ╰╮
 7.08 ┤      │
 7.08 ┤      ╰─╮
 7.08 ┤        ╰──╮
 7.08 ┤           ╰────
                    Daily Average Rates (2025-11-23 to 2025-11-25)

 0.0077 ┼──╮
 0.0076 ┤  │
 0.0075 ┤  │
 0.0074 ┤  ╰╮
 0.0073 ┤   │
 0.0072 ┤   │
 0.0071 ┤   │
 0.0070 ┤   ╰╮
 0.0069 ┤    │
 0.0068 ┤    │
 0.0067 ┤    ╰───
                    Daily Volatility (2025-11-23 to 2025-11-25)
```

### Pattern Analysis

Discover historical patterns to identify optimal times for currency exchange:

```bash
# Analyze last 30 days (default)
./ratemon patterns

# Analyze specific time window
./ratemon patterns --days 60 --weeks 8

# Shorter analysis period
./ratemon patterns --days 7 --weeks 2
```

**Example Output:**

```
Exchange Rate Patterns Analysis
═══════════════════════════════
Analyzing last 30 days of data

Hourly Patterns (Business Hours 08:00-22:00 CST)
─────────────────────────────────────────────────

Hour      Avg Rate    Min         Max         Samples   Peak Freq
───────────────────────────────────────────────────────────────────────────
08:00        7.0725      7.0698      7.0755       850    2 ( 6.7%)
09:00        7.0738      7.0702      7.0778       860    5 (16.7%)
10:00        7.0745      7.0710      7.0789       865    4 (13.3%)
11:00        7.0752      7.0715      7.0795       870    3 (10.0%)
12:00        7.0758      7.0720      7.0802       875    6 (20.0%) 🏆 Most peaks
13:00        7.0761      7.0722      7.0805       880    5 (16.7%)
14:00        7.0763      7.0725      7.0810       885    7 (23.3%) ⭐ Highest avg
15:00        7.0758      7.0720      7.0800       882    2 ( 6.7%)
16:00        7.0750      7.0712      7.0785       878    1 ( 3.3%)
...

Key Insights:
  • Highest average rate: 14:00 (7.0763 CNY)
  • Peak time: 14:00 (7/30 days = 23.3%)
  • Most volatile hour: 14:00 (range: 0.0085 CNY)

Day of Week Patterns (Last 4 weeks)
────────────────────────────────────

Day         Avg Rate    Min         Max         Avg Range     Days
───────────────────────────────────────────────────────────────────────────
Sunday          7.0720      7.0698      7.0750      0.0052           4
Monday          7.0735      7.0705      7.0775      0.0070           4
Tuesday         7.0748      7.0715      7.0790      0.0075           5 ⭐ Best
Wednesday       7.0745      7.0710      7.0785      0.0075           4
Thursday        7.0738      7.0702      7.0780      0.0078           5
Friday          7.0730      7.0695      7.0770      0.0075           4
Saturday        7.0715      7.0685      7.0745      0.0060           4 ↓ Lowest

Weekly Insights:
  • Best day: Tuesday (avg 7.0748 CNY)
  • Lowest day: Saturday (avg 7.0715 CNY)
  • Weekly variance: 0.0033 CNY
  • Most volatile day: Thursday (avg range: 0.0078 CNY)
```

**Use Cases:**

- **Optimal Timing**: Identify hours when rates are historically highest
- **Trend Analysis**: Understand weekly patterns to plan currency exchanges
- **Risk Assessment**: See which hours/days have highest volatility
- **Predictive Insights**: Use historical frequency to estimate when peaks occur

### Data Retention Management

Manage storage efficiently with automatic data aggregation and retention:

```bash
# Show current retention statistics
./ratemon retention --stats

# Preview what retention would do (dry run)
./ratemon retention --dry-run

# Execute retention policy (aggregate and delete old data)
./ratemon retention

# Custom retention periods
./ratemon retention --raw-days 60 --hourly-days 180
```

**Retention Strategy:**

The retention system implements a three-tier approach that balances storage efficiency with predictive capability:

| Data Tier | Granularity | Retention | Purpose | Storage |
|-----------|-------------|-----------|---------|---------|
| **Raw** | 1 minute | 90 days | Recent alerts, change detection | ~51 MB |
| **Hourly** | 1 hour | 365 days | Pattern analysis, trends | ~200 KB |
| **Daily** | 1 day | Forever | Long-term trends, history | ~10 KB/year |

**Benefits:**
- ✅ 99.7% storage reduction vs keeping all raw data
- ✅ Maintains full prediction accuracy
- ✅ Quarterly pattern analysis from 90 days of minute-level data
- ✅ Annual trend analysis from 365 days of hourly data
- ✅ Unlimited historical daily averages

**Example Output (stats):**

```
Data Retention Statistics
═════════════════════════

Raw Data (Minute-level):
  Records:     21,450
  Oldest date: 2025-08-28
  Data age:    90 days

Hourly Aggregates:
  Records:     8,760
  Oldest date: 2024-11-26
  Data age:    365 days

Daily Aggregates:
  Records:     730
  Oldest date: 2023-11-26
  Data age:    730 days

Total Records: 30,940
Estimated Size: ~21 MB
Storage Saved:  ~1.2 GB (98.3% reduction)
```

**When to Run:**

The retention policy should be run periodically (e.g., weekly or monthly) to keep storage optimized:

```bash
# Add to crontab (run weekly on Sunday at 2 AM)
0 2 * * 0 cd /path/to/usd-buy-rate-monitor && ./ratemon retention

# Or manually when needed
./ratemon retention --stats  # Check first
./ratemon retention          # Execute
```

**Options:**

- `--stats` - Show current retention statistics only
- `--dry-run` - Preview changes without modifying data
- `--raw-days N` - Keep raw minute-level data for N days (default: 90)
- `--hourly-days N` - Keep hourly aggregates for N days (default: 365)

### Stop the Daemon

Press `Ctrl+C` to stop the daemon gracefully. The poller will finish the current operation and shut down cleanly.

### WeChat Work Notifications Setup (企业微信通知)

Get exchange rate alerts sent directly to your WeChat Work group chat in Chinese.

**Step 1: Create a Group Robot in WeChat Work**

1. Open WeChat Work (企业微信) on your phone or desktop
2. Go to your group chat where you want to receive alerts
3. Click on group settings (⋯) → "Group Robots" (群机器人) → "Add Robot" (添加机器人)
4. Name the robot (e.g., "汇率监控" / Rate Monitor)
5. Copy the Webhook URL (it looks like: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXXXXXXX`)

**Step 2: Install Daemon with WeChat Notifications (macOS)**

Use the installation script with your webhook URL:

```bash
cd /path/to/usd-buy-rate-monitor

# Simple installation with WeChat notifications
./scripts/install-macos-wechat.sh 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY'

# With alert thresholds
./scripts/install-macos-wechat.sh \
  'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY' \
  7.15 \    # Alert when rate exceeds 7.15
  6.95 \    # Alert when rate drops below 6.95
  0.5       # Alert on 0.5% change
```

**Step 3: Verify Setup**

The daemon will send a WeChat notification when alerts are triggered. You can test by:

```bash
# Check daemon is running
launchctl list | grep ratemon

# View logs to confirm WeChat is enabled
tail -f logs/ratemon.log
# Should show: "WeChat notifications enabled"
```

**Alert Message Examples (in Chinese):**

When rate exceeds threshold:
```
【汇率提醒】汇率突破上限
📈 当前汇率：7.1025 CNY
⚠️ 设定上限：7.10 CNY
🕐 触发时间：2025-11-26 14:30:15
```

When rate drops below threshold:
```
【汇率提醒】汇率跌破下限
📉 当前汇率：6.9450 CNY
⚠️ 设定下限：6.95 CNY
🕐 触发时间：2025-11-26 16:20:30
```

When rate changes rapidly:
```
【汇率提醒】汇率快速上涨
📊 当前汇率：7.0850 CNY
📈 涨幅：+0.52%
🕐 触发时间：2025-11-26 10:15:45
```

**Troubleshooting:**

- **No notifications received**: Check webhook URL is correct in the plist file
- **Error in logs**: Verify the WeChat group robot is still active
- **Wrong language**: Messages are automatically sent in Chinese for better readability

## Project Structure

```
usd-buy-rate-monitor/
├── cmd/ratemon/              # Main CLI entry point
│   └── main.go
├── internal/
│   ├── api/                  # CMB API client
│   │   ├── client.go        # HTTP client with retry logic
│   │   └── models.go        # API response models
│   ├── cli/                  # CLI command implementations
│   │   ├── monitor.go       # Monitor command
│   │   ├── history.go       # History command
│   │   ├── peak.go          # Peak analysis command
│   │   ├── average.go       # Average calculation command
│   │   ├── patterns.go      # Pattern analysis command
│   │   └── common.go        # Common utilities
│   ├── storage/              # Data persistence layer
│   │   ├── db.go            # Database connection
│   │   └── repository.go    # Data access methods
│   └── poller/               # Background polling service
│       └── poller.go
├── pkg/
│   └── chart/                # Chart visualization
│       └── chart.go         # ASCII chart rendering
├── migrations/               # SQL schema migrations
│   └── 001_initial_schema.sql
├── data/                     # Database files (gitignored)
├── go.mod
└── README.md
```

## How It Works

1. **API Client**: Fetches exchange rate data from `https://m.cmbchina.com/api/rate/fx-rate`

   - Implements exponential backoff retry logic for reliability
   - Handles network errors and API failures gracefully

2. **Data Extraction**: Parses the CMB API response to extract USD rate

   - Identifies USD currency by Chinese name "美元"
   - Extracts `rtcBid` field and divides by 100 (rate is per 10 units)

3. **Business Hours Check**: Optimizes resource usage by respecting CMB operating hours

   - Default hours: 08:00-22:00 CST (China Standard Time, UTC+8)
   - Skips API calls outside business hours since rates don't update
   - Reduces API calls by ~60% and saves database writes
   - Can be disabled with `--no-business-hours` flag

4. **Storage**: Saves rates to SQLite database

   - Each record includes: rate value, timestamp, and date partition
   - Indexed by date and time for efficient queries

5. **Polling Loop**: Runs continuously with configurable interval
   - Uses `time.Ticker` for precise timing
   - Continues operation even if individual polls fail

## Database Schema

```sql
CREATE TABLE exchange_rates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    currency_code TEXT NOT NULL DEFAULT 'USD',
    rtc_bid REAL NOT NULL,
    collected_at TIMESTAMP NOT NULL,
    date_partition TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX idx_rates_date_time ON exchange_rates(date_partition, collected_at);
CREATE INDEX idx_rates_collected ON exchange_rates(collected_at);
```

## Troubleshooting

**Database locked errors:**

- The database uses WAL mode to reduce lock contention
- Ensure only one daemon instance is running at a time

**API errors:**

- Check your internet connection
- The client automatically retries failed requests with exponential backoff
- Check logs for specific error messages

**Permissions issues:**

- Ensure the `data/` directory is writable
- On Linux/Mac: `chmod 755 data/`

**No data available:**

- Make sure the daemon is running and has collected some data
- Check that the database path is correct
- Verify data with: `sqlite3 ./data/rates.db "SELECT COUNT(*) FROM exchange_rates;"`

## Available Commands

| Command     | Description                                      |
| ----------- | ------------------------------------------------ |
| `daemon`    | Run background polling service                   |
| `monitor`   | Display current/latest exchange rate             |
| `history`   | Query historical rates by time range             |
| `peak`      | Show daily peak exchange rates                   |
| `average`   | Calculate daily average rates                    |
| `patterns`  | Analyze hourly and weekly rate patterns          |
| `retention` | Manage data retention and aggregation            |

Run `./ratemon <command> --help` for detailed usage of each command.

## Future Enhancements

Potential features for future releases:

- Enhanced notifications (email, Slack integration)
- Web dashboard for browser-based monitoring
- Export to Excel format
- Multi-currency support
- Systemd service configuration for Linux servers
- Automated retention scheduling via daemon flag

## License

MIT License
