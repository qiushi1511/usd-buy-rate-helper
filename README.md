# USD/CNY Exchange Rate Monitor

A lightweight monitoring system that tracks the USD/CNY exchange rate from China Merchants Bank's API and provides historical analysis.

## Features

- **Background Polling**: Continuously polls CMB API every minute for USD exchange rates
- **SQLite Storage**: Stores historical data locally with date-based partitioning
- **Real-time Monitoring**: Display current exchange rate with live updates
- **Historical Analysis**: Query rates by time range with multiple output formats
- **Daily Statistics**: Peak rate analysis and average calculations
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
- `-d, --db string` - Database file path (default: ./data/rates.db)
- `-m, --migrations string` - Migrations directory path (default: ./migrations)
- `-v, --verbose` - Enable verbose logging

**Example:**
```bash
# Poll every 30 seconds with verbose logging
./ratemon daemon -i 30s -v

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

### Stop the Daemon

Press `Ctrl+C` to stop the daemon gracefully. The poller will finish the current operation and shut down cleanly.

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

3. **Storage**: Saves rates to SQLite database
   - Each record includes: rate value, timestamp, and date partition
   - Indexed by date and time for efficient queries

4. **Polling Loop**: Runs continuously with configurable interval
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

| Command | Description |
|---------|-------------|
| `daemon` | Run background polling service |
| `monitor` | Display current/latest exchange rate |
| `history` | Query historical rates by time range |
| `peak` | Show daily peak exchange rates |
| `average` | Calculate daily average rates |

Run `./ratemon <command> --help` for detailed usage of each command.

## Future Enhancements

Potential features for future releases:
- Data retention and aggregation (30-day policy)
- Alert/notification system for threshold monitoring
- Web dashboard for browser-based monitoring
- Export to Excel format
- Multi-currency support
- Systemd service configuration for production deployment

## License

MIT License
