-- Initial schema for USD/CNY exchange rate monitoring
-- Creates the core table for storing minute-level exchange rate data

CREATE TABLE IF NOT EXISTS exchange_rates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    currency_code TEXT NOT NULL DEFAULT 'USD',
    rtc_bid REAL NOT NULL,
    collected_at TIMESTAMP NOT NULL,
    date_partition TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (rtc_bid > 0),
    CHECK (currency_code IN ('USD'))
);

-- Index for efficient date-based queries
CREATE INDEX IF NOT EXISTS idx_rates_date_time
    ON exchange_rates(date_partition, collected_at);

-- Index for time-range queries
CREATE INDEX IF NOT EXISTS idx_rates_collected
    ON exchange_rates(collected_at);
