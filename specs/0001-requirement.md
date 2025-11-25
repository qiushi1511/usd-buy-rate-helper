# USD/CNY Exchange Rate Monitoring System

**Objective**: Build a lightweight monitoring system that tracks the USD (美元) rtcBid exchange rate from China Merchants Bank's API and provides historical analysis through a command-line interface.

**Core Functionality**

The system continuously polls `https://m.cmbchina.com/api/rate/fx-rate` every minute to extract the rtcBid value for USD. Each data point includes the rate value along with its timestamp. The collected data is organized by date and persisted to local storage in a structured format (consider JSON or SQLite for easy querying).

**Data Storage Strategy**

Historical data should be partitioned by date to facilitate efficient queries and prevent individual files from growing too large. Each day's data file contains all minute-level readings for that date. The system maintains an index or manifest file for quick date lookups. Consider implementing automatic data retention policies (e.g., keep raw minute-level data for 30 days, then aggregate to hourly averages for longer-term storage).

**CLI Interface Requirements**

The command-line interface provides four primary views:

1. **Real-time monitoring** - Display the current rate with timestamp, update frequency, and comparison to previous reading (showing the delta)

2. **Time-based history** - Query rates for a specific time range (e.g., "show me rates between 9:00 AM and 5:00 PM today" or "show last 2 hours")

3. **Daily peak analysis** - Display the highest rate recorded for any given date, including the exact timestamp when it occurred

4. **Daily average** - Calculate and display the mean rate for specified dates, with options to compare across multiple dates

**Additional Considerations**

Think about error handling for API failures (network issues, rate limiting, API changes). You might want to implement retry logic with exponential backoff and maintain a log of failed requests. Also consider adding alerting capabilities (e.g., notify when rate crosses certain thresholds) and basic visualization like ASCII charts for trend viewing directly in the terminal.

For the tech stack, this could be built as a Go CLI tool with data persistence in SQLite for robust querying capabilities and easy local storage.
