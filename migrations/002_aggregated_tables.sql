-- Migration: Create aggregated tables for data retention
-- This enables efficient long-term storage while maintaining prediction accuracy

-- Hourly aggregated rates (365 days retention)
CREATE TABLE IF NOT EXISTS hourly_rates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date_partition TEXT NOT NULL,      -- YYYY-MM-DD
    hour INTEGER NOT NULL,              -- 0-23
    avg_rate REAL NOT NULL,
    min_rate REAL NOT NULL,
    max_rate REAL NOT NULL,
    sample_count INTEGER NOT NULL,
    first_collected_at TEXT NOT NULL,
    last_collected_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(date_partition, hour)
);

-- Daily aggregated rates (permanent retention)
CREATE TABLE IF NOT EXISTS daily_rates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date_partition TEXT NOT NULL UNIQUE,
    avg_rate REAL NOT NULL,
    min_rate REAL NOT NULL,
    max_rate REAL NOT NULL,
    peak_rate REAL NOT NULL,           -- Highest rate of the day
    peak_time TEXT NOT NULL,           -- When peak occurred
    volatility REAL NOT NULL,          -- max_rate - min_rate
    sample_count INTEGER NOT NULL,
    first_collected_at TEXT NOT NULL,
    last_collected_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_hourly_date ON hourly_rates(date_partition);
CREATE INDEX IF NOT EXISTS idx_hourly_date_hour ON hourly_rates(date_partition, hour);
CREATE INDEX IF NOT EXISTS idx_daily_date ON daily_rates(date_partition);

-- Create view for easy access to all data (raw + aggregated)
CREATE VIEW IF NOT EXISTS all_rates AS
-- Recent raw data (last 90 days)
SELECT
    'raw' as source,
    collected_at as timestamp,
    rtc_bid as rate,
    date_partition
FROM exchange_rates
WHERE date_partition >= date('now', '-90 days')

UNION ALL

-- Hourly aggregates (91-365 days ago)
SELECT
    'hourly' as source,
    datetime(date_partition || ' ' || printf('%02d', hour) || ':00:00') as timestamp,
    avg_rate as rate,
    date_partition
FROM hourly_rates
WHERE date_partition < date('now', '-90 days')
  AND date_partition >= date('now', '-365 days')

UNION ALL

-- Daily aggregates (older than 365 days)
SELECT
    'daily' as source,
    datetime(date_partition || ' 12:00:00') as timestamp,
    avg_rate as rate,
    date_partition
FROM daily_rates
WHERE date_partition < date('now', '-365 days')

ORDER BY timestamp DESC;
