# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a USD/CNY exchange rate monitoring system that continuously polls China Merchants Bank's API and provides historical analysis, pattern detection, and intelligent alerts via WeChat Work notifications. The goal is to help users identify optimal times to exchange RMB to USD by analyzing historical patterns and providing real-time notifications.

## Build and Test Commands

```bash
# Build the binary
go build -o ratemon ./cmd/ratemon

# Run tests
go test ./...

# Run specific package tests
go test ./internal/poller/...

# Run with verbose output
go test -v ./...

# Build and run daemon (development)
go build -o ratemon ./cmd/ratemon && ./ratemon daemon -v

# Quick validation (check if it compiles and shows help)
go build -o ratemon ./cmd/ratemon && ./ratemon --help
```

## Architecture Overview

### Three-Tier Data Model

The system implements a **multi-tier retention strategy** that balances storage efficiency with predictive accuracy:

1. **Raw Data (90 days)**: Minute-level granularity stored in `exchange_rates` table
2. **Hourly Aggregates (365 days)**: Aggregated stats in `hourly_rates` table
3. **Daily Aggregates (Forever)**: Long-term trends in `daily_rates` table

This achieves 99.7% storage reduction while maintaining full prediction capability. The retention system automatically aggregates old raw data before deletion.

### Core Components

**Polling Layer** (`internal/poller/`):
- Manages background data collection with configurable intervals
- Implements **business hours optimization** (08:30-22:00 CST) - skips polling when CMB rates don't update, reducing API calls by ~60%
- Integrates with alert system to trigger notifications on rate changes
- Uses `time.Ticker` for precise timing and handles graceful shutdown

**Alert System** (`internal/alerts/`):
- **Manager**: Evaluates rates against configured thresholds and patterns
- **Notifiers**: Pluggable notification backends (Log, WeChat Work)
- **Cooldown mechanism**: Prevents alert spam by tracking last notification time per alert type
- **Pattern detection**: Compares current rates against historical averages using standard deviation

**Data Access** (`internal/storage/`):
- **DB**: SQLite connection management with WAL mode for concurrency
- **Repository**: Pattern-based data access methods (GetLatest, GetByTimeRange, GetPeak, GetAverage, GetHourlyPatterns, GetDayOfWeekPatterns)
- **Migration system**: Numbered SQL files (001_*.sql, 002_*.sql) applied in order

**API Client** (`internal/api/`):
- CMB API integration with exponential backoff retry (max 3 attempts)
- Extracts USD rate from response by finding currency with name "美元"
- Rate conversion: divides `rtcBid` by 100 (CMB returns rate per 10 units)
- Structured error types: `NetworkError`, `HTTPError`

**CLI Commands** (`internal/cli/`):
- Each command is a separate file: monitor.go, history.go, peak.go, average.go, patterns.go, retention.go
- Uses Cobra framework for command structure defined in `cmd/ratemon/main.go`
- Chart rendering via `pkg/chart/` using asciigraph library

### Time Zone Handling

**Critical**: The system operates in two time zones:
- **CST (UTC+8)**: For business hours checking (CMB operates in China)
- **Local time**: For data storage and user-facing displays

When checking business hours, always convert to CST using:
```go
cstLocation := time.FixedZone("CST", 8*60*60)
now := time.Now().In(cstLocation)
```

### Alert Architecture

Alerts follow a **check-and-notify** pattern:

1. **Manager.Check()**: Evaluates current rate, returns list of triggered alerts
2. **Notifier.Notify()**: Sends alert via specific channel (log, WeChat, etc.)
3. **Cooldown tracking**: Manager stores last notification time to prevent spam

WeChat notifications are formatted in Chinese with emojis for better mobile readability.

## Database Schema

### Core Tables

**exchange_rates**: Raw minute-level data
- Indexed by `(date_partition, collected_at)` for efficient time-range queries
- `date_partition` enables fast date filtering without timestamp parsing

**hourly_rates**: Aggregated hourly statistics
- Created by `AggregateToHourly()` from raw data
- Stores avg/min/max/sample_count per hour

**daily_rates**: Aggregated daily statistics
- Created by `AggregateToDaily()` from raw data
- Includes peak rate, peak time, and volatility metrics

**all_rates view**: Unified view combining all three tiers based on date ranges

### Migration System

Migrations are numbered SQL files executed in alphanumeric order:
- `001_initial_schema.sql` - Core tables and indexes
- `002_aggregated_tables.sql` - Retention system tables

When adding new migrations, use the next number in sequence (003_*.sql).

## WeChat Work Integration

WeChat notifications require a webhook URL from WeChat Work (企业微信) group chat robots. The webhook format is:
```
https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXXXXXXX
```

Messages are formatted in Chinese with specific emojis for each alert type:
- 📈 High threshold breach
- 📉 Low threshold breach
- 📊 Rapid change
- ⚠️ Unusual pattern

The `formatChineseMessage()` method in `alerts/alerts.go` handles message formatting.

## Development Patterns

### Adding a New CLI Command

1. Create command file in `internal/cli/` (e.g., `my_command.go`)
2. Implement command struct with `DisplayXXX()` method
3. Add constructor in `cmd/ratemon/main.go`:
   ```go
   func newMyCmd() *cobra.Command { ... }
   func runMy(ctx context.Context, ...) error { ... }
   ```
4. Register in main(): `rootCmd.AddCommand(newMyCmd())`

### Adding a New Alert Type

1. Define new `AlertType` constant in `internal/alerts/alerts.go`
2. Add check logic in `Manager.Check()` method
3. Update `formatChineseMessage()` for WeChat formatting
4. Add corresponding flag in daemon command if needed

### Extending Repository Methods

Pattern to follow:
```go
func (r *Repository) GetXXX(ctx context.Context, params...) (ResultType, error) {
    query := `SELECT ... FROM exchange_rates WHERE ...`
    rows, err := r.db.conn.QueryContext(ctx, query, params...)
    if err != nil {
        return nil, fmt.Errorf("querying XXX: %w", err)
    }
    defer rows.Close()

    // Scan and process results
    // Return wrapped errors with context
}
```

Always use `QueryContext` or `ExecContext` to support cancellation.

## Common Gotchas

**SQLite NULL handling**: When querying MIN/MAX on potentially empty tables, use `sql.NullString`:
```go
var oldest sql.NullString
err = db.QueryRow("SELECT MIN(date) FROM table").Scan(&oldest)
if oldest.Valid {
    // Use oldest.String
}
```

**Business hours in tests**: When testing poller behavior, remember the business hours check uses CST (UTC+8), not local time.

**Rate conversion**: CMB API returns rates per 10 units, always divide by 100:
```go
actualRate := apiResponse.RtcBid / 100.0
```

**Date partition format**: Always use `"2006-01-02"` format for date_partition to ensure correct string sorting and comparisons.

## Deployment

The system supports macOS LaunchAgent deployment via `scripts/install-macos-wechat.sh`. This script:
- Builds the binary
- Creates LaunchAgent plist with WeChat webhook configuration
- Sets up auto-start and auto-restart
- Configures logging to `logs/ratemon.log`

For production deployment, ensure:
1. Database path is writable
2. Migrations directory is accessible
3. WeChat webhook URL is valid (if using notifications)
4. Sufficient disk space for retention strategy (90 days raw ~51MB)

## Project Goals

The primary goal is to **help users exchange RMB to USD at optimal rates** by:
1. Identifying historical patterns (best hours/days for high rates)
2. Providing real-time alerts when rates reach favorable levels
3. Analyzing trends to predict optimal timing
4. Minimizing monitoring effort via automated notifications

Future enhancements should focus on **smart recommendations** (when to exchange vs. wait) and **predictive modeling** based on historical patterns.
