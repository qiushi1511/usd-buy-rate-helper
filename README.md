# USD/CNY Exchange Rate Monitor

A lightweight monitoring system that tracks the USD/CNY exchange rate from China Merchants Bank's API and provides historical analysis.

## Features (MVP)

- **Background Polling**: Continuously polls CMB API every minute for USD exchange rates
- **SQLite Storage**: Stores historical data locally with date-based partitioning
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

### Verify Data Collection

While the daemon is running, you can directly query the SQLite database to verify data collection:

```bash
sqlite3 ./data/rates.db "SELECT * FROM exchange_rates ORDER BY collected_at DESC LIMIT 10;"
```

Or check the total number of records:

```bash
sqlite3 ./data/rates.db "SELECT COUNT(*) FROM exchange_rates;"
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
│   ├── storage/             # Data persistence layer
│   │   ├── db.go            # Database connection
│   │   └── repository.go    # Data access methods
│   └── poller/              # Background polling service
│       └── poller.go
├── migrations/              # SQL schema migrations
│   └── 001_initial_schema.sql
├── data/                    # Database files (gitignored)
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

## Coming Soon

Additional CLI commands for data analysis (in development):
- `ratemon monitor` - Real-time rate display
- `ratemon history` - Query historical data
- `ratemon peak` - Daily peak analysis
- `ratemon average` - Daily average calculation

## License

MIT License
