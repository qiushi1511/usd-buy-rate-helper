package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// ExchangeRate represents a single exchange rate record
type ExchangeRate struct {
	ID            int64
	CurrencyCode  string
	RtcBid        float64
	CollectedAt   time.Time
	DatePartition string
	CreatedAt     time.Time
}

// Repository provides data access methods for exchange rates
type Repository struct {
	db     *DB
	logger *slog.Logger
}

// NewRepository creates a new repository instance
func NewRepository(db *DB, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// InsertRate stores a new exchange rate reading
func (r *Repository) InsertRate(ctx context.Context, rate *ExchangeRate) error {
	query := `
		INSERT INTO exchange_rates (currency_code, rtc_bid, collected_at, date_partition)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.conn.ExecContext(ctx, query,
		rate.CurrencyCode,
		rate.RtcBid,
		rate.CollectedAt,
		rate.DatePartition,
	)
	if err != nil {
		return fmt.Errorf("inserting rate: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting insert ID: %w", err)
	}

	rate.ID = id
	return nil
}

// GetLatestRate retrieves the most recent exchange rate
func (r *Repository) GetLatestRate(ctx context.Context) (*ExchangeRate, error) {
	query := `
		SELECT id, currency_code, rtc_bid, collected_at, date_partition, created_at
		FROM exchange_rates
		ORDER BY collected_at DESC
		LIMIT 1
	`

	var rate ExchangeRate
	err := r.db.conn.QueryRowContext(ctx, query).Scan(
		&rate.ID,
		&rate.CurrencyCode,
		&rate.RtcBid,
		&rate.CollectedAt,
		&rate.DatePartition,
		&rate.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying latest rate: %w", err)
	}

	return &rate, nil
}

// GetRatesByTimeRange retrieves rates within a time range
func (r *Repository) GetRatesByTimeRange(ctx context.Context, start, end time.Time) ([]ExchangeRate, error) {
	query := `
		SELECT id, currency_code, rtc_bid, collected_at, date_partition, created_at
		FROM exchange_rates
		WHERE collected_at >= ? AND collected_at <= ?
		ORDER BY collected_at ASC
	`

	rows, err := r.db.conn.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("querying rates: %w", err)
	}
	defer rows.Close()

	var rates []ExchangeRate
	for rows.Next() {
		var rate ExchangeRate
		err := rows.Scan(
			&rate.ID,
			&rate.CurrencyCode,
			&rate.RtcBid,
			&rate.CollectedAt,
			&rate.DatePartition,
			&rate.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning rate: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rates: %w", err)
	}

	return rates, nil
}

// GetDailyPeak finds the highest rate for a given date
func (r *Repository) GetDailyPeak(ctx context.Context, date string) (*ExchangeRate, error) {
	query := `
		SELECT id, currency_code, rtc_bid, collected_at, date_partition, created_at
		FROM exchange_rates
		WHERE date_partition = ?
		ORDER BY rtc_bid DESC
		LIMIT 1
	`

	var rate ExchangeRate
	err := r.db.conn.QueryRowContext(ctx, query, date).Scan(
		&rate.ID,
		&rate.CurrencyCode,
		&rate.RtcBid,
		&rate.CollectedAt,
		&rate.DatePartition,
		&rate.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying daily peak: %w", err)
	}

	return &rate, nil
}

// DailyStats contains aggregate statistics for a date
type DailyStats struct {
	Date        string
	MinRate     float64
	MaxRate     float64
	AvgRate     float64
	PeakTime    time.Time
	SampleCount int
}

// GetDailyStats calculates aggregate statistics for a date
func (r *Repository) GetDailyStats(ctx context.Context, date string) (*DailyStats, error) {
	query := `
		SELECT
			MIN(rtc_bid) as min_rate,
			MAX(rtc_bid) as max_rate,
			AVG(rtc_bid) as avg_rate,
			COUNT(*) as sample_count
		FROM exchange_rates
		WHERE date_partition = ?
	`

	var stats DailyStats
	stats.Date = date

	err := r.db.conn.QueryRowContext(ctx, query, date).Scan(
		&stats.MinRate,
		&stats.MaxRate,
		&stats.AvgRate,
		&stats.SampleCount,
	)
	if err != nil {
		return nil, fmt.Errorf("querying daily stats: %w", err)
	}

	// Get peak time separately
	peakQuery := `
		SELECT collected_at
		FROM exchange_rates
		WHERE date_partition = ? AND rtc_bid = ?
		LIMIT 1
	`

	err = r.db.conn.QueryRowContext(ctx, peakQuery, date, stats.MaxRate).Scan(&stats.PeakTime)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("querying peak time: %w", err)
	}

	return &stats, nil
}

// Count returns the total number of exchange rate records
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM exchange_rates").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting records: %w", err)
	}
	return count, nil
}

// HourlyPattern represents statistics for a specific hour of the day
type HourlyPattern struct {
	Hour        int
	AvgRate     float64
	MinRate     float64
	MaxRate     float64
	SampleCount int
	PeakFreq    int // How many times this hour had the daily peak
}

// GetHourlyPatterns analyzes rate patterns by hour of day over the last N days
func (r *Repository) GetHourlyPatterns(ctx context.Context, days int) ([]HourlyPattern, error) {
	query := `
		SELECT
			CAST(strftime('%H', collected_at) AS INTEGER) as hour,
			AVG(rtc_bid) as avg_rate,
			MIN(rtc_bid) as min_rate,
			MAX(rtc_bid) as max_rate,
			COUNT(*) as sample_count
		FROM exchange_rates
		WHERE date_partition >= date('now', '-' || ? || ' days')
		GROUP BY hour
		ORDER BY hour
	`

	rows, err := r.db.conn.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("querying hourly patterns: %w", err)
	}
	defer rows.Close()

	var patterns []HourlyPattern
	for rows.Next() {
		var p HourlyPattern
		err := rows.Scan(
			&p.Hour,
			&p.AvgRate,
			&p.MinRate,
			&p.MaxRate,
			&p.SampleCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning hourly pattern: %w", err)
		}
		patterns = append(patterns, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hourly patterns: %w", err)
	}

	// Calculate peak frequency for each hour
	for i := range patterns {
		freq, err := r.getHourPeakFrequency(ctx, patterns[i].Hour, days)
		if err != nil {
			r.logger.Warn("failed to get peak frequency", "hour", patterns[i].Hour, "error", err)
		} else {
			patterns[i].PeakFreq = freq
		}
	}

	return patterns, nil
}

// getHourPeakFrequency counts how many times a given hour had the daily peak
func (r *Repository) getHourPeakFrequency(ctx context.Context, hour, days int) (int, error) {
	query := `
		WITH daily_peaks AS (
			SELECT
				date_partition,
				MAX(rtc_bid) as peak_rate
			FROM exchange_rates
			WHERE date_partition >= date('now', '-' || ? || ' days')
			GROUP BY date_partition
		)
		SELECT COUNT(DISTINCT e.date_partition)
		FROM exchange_rates e
		INNER JOIN daily_peaks dp ON e.date_partition = dp.date_partition AND e.rtc_bid = dp.peak_rate
		WHERE CAST(strftime('%H', e.collected_at) AS INTEGER) = ?
	`

	var count int
	err := r.db.conn.QueryRowContext(ctx, query, days, hour).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("querying peak frequency: %w", err)
	}

	return count, nil
}

// DayOfWeekPattern represents statistics for a specific day of the week
type DayOfWeekPattern struct {
	DayOfWeek   int    // 0=Sunday, 1=Monday, etc.
	DayName     string
	AvgRate     float64
	MinRate     float64
	MaxRate     float64
	AvgRange    float64
	SampleDays  int
}

// GetDayOfWeekPatterns analyzes rate patterns by day of week
func (r *Repository) GetDayOfWeekPatterns(ctx context.Context, weeks int) ([]DayOfWeekPattern, error) {
	query := `
		WITH daily_data AS (
			SELECT
				date_partition,
				strftime('%w', collected_at) as dow,
				AVG(rtc_bid) as avg_rate,
				MIN(rtc_bid) as min_rate,
				MAX(rtc_bid) as max_rate,
				(MAX(rtc_bid) - MIN(rtc_bid)) as range
			FROM exchange_rates
			WHERE date_partition >= date('now', '-' || ? || ' days')
			GROUP BY date_partition
		)
		SELECT
			CAST(dow AS INTEGER) as day_of_week,
			AVG(avg_rate) as avg_rate,
			MIN(min_rate) as min_rate,
			MAX(max_rate) as max_rate,
			AVG(range) as avg_range,
			COUNT(*) as sample_days
		FROM daily_data
		GROUP BY dow
		ORDER BY day_of_week
	`

	rows, err := r.db.conn.QueryContext(ctx, query, weeks*7)
	if err != nil {
		return nil, fmt.Errorf("querying day of week patterns: %w", err)
	}
	defer rows.Close()

	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	var patterns []DayOfWeekPattern
	for rows.Next() {
		var p DayOfWeekPattern
		err := rows.Scan(
			&p.DayOfWeek,
			&p.AvgRate,
			&p.MinRate,
			&p.MaxRate,
			&p.AvgRange,
			&p.SampleDays,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning day of week pattern: %w", err)
		}
		p.DayName = dayNames[p.DayOfWeek]
		patterns = append(patterns, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating day of week patterns: %w", err)
	}

	return patterns, nil
}

// HourlyRate represents aggregated hourly statistics
type HourlyRate struct {
	ID                int64
	DatePartition     string
	Hour              int
	AvgRate           float64
	MinRate           float64
	MaxRate           float64
	SampleCount       int
	FirstCollectedAt  time.Time
	LastCollectedAt   time.Time
	CreatedAt         time.Time
}

// DailyRate represents aggregated daily statistics
type DailyRate struct {
	ID                int64
	DatePartition     string
	AvgRate           float64
	MinRate           float64
	MaxRate           float64
	PeakRate          float64
	PeakTime          time.Time
	Volatility        float64
	SampleCount       int
	FirstCollectedAt  time.Time
	LastCollectedAt   time.Time
	CreatedAt         time.Time
}

// AggregateToHourly creates hourly aggregates from raw data for a specific date
func (r *Repository) AggregateToHourly(ctx context.Context, datePartition string) (int, error) {
	query := `
		INSERT OR REPLACE INTO hourly_rates (
			date_partition, hour, avg_rate, min_rate, max_rate, sample_count,
			first_collected_at, last_collected_at
		)
		SELECT
			date_partition,
			CAST(strftime('%H', collected_at) AS INTEGER) as hour,
			AVG(rtc_bid) as avg_rate,
			MIN(rtc_bid) as min_rate,
			MAX(rtc_bid) as max_rate,
			COUNT(*) as sample_count,
			MIN(collected_at) as first_collected_at,
			MAX(collected_at) as last_collected_at
		FROM exchange_rates
		WHERE date_partition = ?
		GROUP BY date_partition, hour
	`

	result, err := r.db.conn.ExecContext(ctx, query, datePartition)
	if err != nil {
		return 0, fmt.Errorf("aggregating to hourly: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return int(rows), nil
}

// AggregateToDaily creates daily aggregates from raw data for a specific date
func (r *Repository) AggregateToDaily(ctx context.Context, datePartition string) error {
	// First, find the peak rate and its time
	var peakRate float64
	var peakTime string
	peakQuery := `
		SELECT rtc_bid, collected_at
		FROM exchange_rates
		WHERE date_partition = ?
		ORDER BY rtc_bid DESC
		LIMIT 1
	`
	err := r.db.conn.QueryRowContext(ctx, peakQuery, datePartition).Scan(&peakRate, &peakTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // No data for this date
		}
		return fmt.Errorf("finding peak rate: %w", err)
	}

	// Then, insert the daily aggregate
	query := `
		INSERT OR REPLACE INTO daily_rates (
			date_partition, avg_rate, min_rate, max_rate, peak_rate, peak_time,
			volatility, sample_count, first_collected_at, last_collected_at
		)
		SELECT
			date_partition,
			AVG(rtc_bid) as avg_rate,
			MIN(rtc_bid) as min_rate,
			MAX(rtc_bid) as max_rate,
			? as peak_rate,
			? as peak_time,
			(MAX(rtc_bid) - MIN(rtc_bid)) as volatility,
			COUNT(*) as sample_count,
			MIN(collected_at) as first_collected_at,
			MAX(collected_at) as last_collected_at
		FROM exchange_rates
		WHERE date_partition = ?
	`

	_, err = r.db.conn.ExecContext(ctx, query, peakRate, peakTime, datePartition)
	if err != nil {
		return fmt.Errorf("aggregating to daily: %w", err)
	}

	return nil
}

// DeleteRawDataBefore deletes raw exchange rate data before a specific date
func (r *Repository) DeleteRawDataBefore(ctx context.Context, beforeDate string) (int64, error) {
	query := `DELETE FROM exchange_rates WHERE date_partition < ?`

	result, err := r.db.conn.ExecContext(ctx, query, beforeDate)
	if err != nil {
		return 0, fmt.Errorf("deleting raw data: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return rows, nil
}

// DeleteHourlyDataBefore deletes hourly aggregates before a specific date
func (r *Repository) DeleteHourlyDataBefore(ctx context.Context, beforeDate string) (int64, error) {
	query := `DELETE FROM hourly_rates WHERE date_partition < ?`

	result, err := r.db.conn.ExecContext(ctx, query, beforeDate)
	if err != nil {
		return 0, fmt.Errorf("deleting hourly data: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return rows, nil
}

// GetOldRawDataDates returns dates that have raw data older than the retention period
func (r *Repository) GetOldRawDataDates(ctx context.Context, retentionDays int) ([]string, error) {
	query := `
		SELECT DISTINCT date_partition
		FROM exchange_rates
		WHERE date_partition < date('now', '-' || ? || ' days')
		ORDER BY date_partition
	`

	rows, err := r.db.conn.QueryContext(ctx, query, retentionDays)
	if err != nil {
		return nil, fmt.Errorf("querying old raw data dates: %w", err)
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, fmt.Errorf("scanning date: %w", err)
		}
		dates = append(dates, date)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating dates: %w", err)
	}

	return dates, nil
}

// RetentionStats holds statistics about data retention
type RetentionStats struct {
	RawRecords    int64
	HourlyRecords int64
	DailyRecords  int64
	OldestRaw     string
	OldestHourly  string
	OldestDaily   string
	TotalSize     int64
}

// GetRetentionStats returns current retention statistics
func (r *Repository) GetRetentionStats(ctx context.Context) (*RetentionStats, error) {
	stats := &RetentionStats{}

	// Count raw records
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM exchange_rates").Scan(&stats.RawRecords)
	if err != nil {
		return nil, fmt.Errorf("counting raw records: %w", err)
	}

	// Count hourly records
	err = r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM hourly_rates").Scan(&stats.HourlyRecords)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("counting hourly records: %w", err)
	}

	// Count daily records
	err = r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM daily_rates").Scan(&stats.DailyRecords)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("counting daily records: %w", err)
	}

	// Get oldest raw date (use sql.NullString to handle NULL)
	var oldestRaw sql.NullString
	err = r.db.conn.QueryRowContext(ctx, "SELECT MIN(date_partition) FROM exchange_rates").Scan(&oldestRaw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting oldest raw date: %w", err)
	}
	if oldestRaw.Valid {
		stats.OldestRaw = oldestRaw.String
	}

	// Get oldest hourly date
	var oldestHourly sql.NullString
	err = r.db.conn.QueryRowContext(ctx, "SELECT MIN(date_partition) FROM hourly_rates").Scan(&oldestHourly)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting oldest hourly date: %w", err)
	}
	if oldestHourly.Valid {
		stats.OldestHourly = oldestHourly.String
	}

	// Get oldest daily date
	var oldestDaily sql.NullString
	err = r.db.conn.QueryRowContext(ctx, "SELECT MIN(date_partition) FROM daily_rates").Scan(&oldestDaily)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting oldest daily date: %w", err)
	}
	if oldestDaily.Valid {
		stats.OldestDaily = oldestDaily.String
	}

	return stats, nil
}
