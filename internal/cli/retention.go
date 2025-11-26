package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// RetentionCommand handles data retention operations
type RetentionCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewRetentionCommand creates a new retention command instance
func NewRetentionCommand(repo *storage.Repository, logger *slog.Logger) *RetentionCommand {
	return &RetentionCommand{
		repo:   repo,
		logger: logger,
	}
}

// Run executes the retention policy
func (r *RetentionCommand) Run(ctx context.Context, rawRetentionDays, hourlyRetentionDays int, dryRun bool) error {
	r.logger.Info("starting retention policy execution",
		"raw_retention_days", rawRetentionDays,
		"hourly_retention_days", hourlyRetentionDays,
		"dry_run", dryRun)

	// Get old raw data dates that need aggregation
	oldDates, err := r.repo.GetOldRawDataDates(ctx, rawRetentionDays)
	if err != nil {
		return fmt.Errorf("getting old raw data dates: %w", err)
	}

	if len(oldDates) == 0 {
		r.logger.Info("no old data to process")
		fmt.Println("No data older than retention period found.")
		return nil
	}

	fmt.Printf("Found %d dates with data older than %d days\n", len(oldDates), rawRetentionDays)
	fmt.Printf("Date range: %s to %s\n\n", oldDates[0], oldDates[len(oldDates)-1])

	if dryRun {
		fmt.Println("DRY RUN MODE - No actual changes will be made\n")
	}

	// Process each date
	totalHourlyCreated := 0
	totalDailyCreated := 0

	for _, date := range oldDates {
		r.logger.Debug("processing date", "date", date)

		if !dryRun {
			// Create hourly aggregates
			hourlyCount, err := r.repo.AggregateToHourly(ctx, date)
			if err != nil {
				r.logger.Error("failed to aggregate hourly", "date", date, "error", err)
				continue
			}
			totalHourlyCreated += hourlyCount

			// Create daily aggregates
			err = r.repo.AggregateToDaily(ctx, date)
			if err != nil {
				r.logger.Error("failed to aggregate daily", "date", date, "error", err)
				continue
			}
			totalDailyCreated++
		}
	}

	if !dryRun {
		fmt.Printf("✅ Created %d hourly aggregates and %d daily aggregates\n\n", totalHourlyCreated, totalDailyCreated)

		// Delete old raw data
		cutoffDate := time.Now().AddDate(0, 0, -rawRetentionDays).Format("2006-01-02")
		deletedRaw, err := r.repo.DeleteRawDataBefore(ctx, cutoffDate)
		if err != nil {
			return fmt.Errorf("deleting old raw data: %w", err)
		}
		fmt.Printf("🗑️  Deleted %d raw records older than %s\n", deletedRaw, cutoffDate)

		// Delete old hourly data
		hourlyCutoffDate := time.Now().AddDate(0, 0, -hourlyRetentionDays).Format("2006-01-02")
		deletedHourly, err := r.repo.DeleteHourlyDataBefore(ctx, hourlyCutoffDate)
		if err != nil {
			return fmt.Errorf("deleting old hourly data: %w", err)
		}
		fmt.Printf("🗑️  Deleted %d hourly records older than %s\n\n", deletedHourly, hourlyCutoffDate)
	} else {
		fmt.Printf("Would create ~%d hourly aggregates and %d daily aggregates\n", len(oldDates)*14, len(oldDates))
		fmt.Printf("Would delete raw data older than %d days\n", rawRetentionDays)
		fmt.Printf("Would delete hourly data older than %d days\n\n", hourlyRetentionDays)
	}

	r.logger.Info("retention policy completed")
	return nil
}

// ShowStats displays current retention statistics
func (r *RetentionCommand) ShowStats(ctx context.Context) error {
	stats, err := r.repo.GetRetentionStats(ctx)
	if err != nil {
		return fmt.Errorf("getting retention stats: %w", err)
	}

	fmt.Println("Data Retention Statistics")
	fmt.Println("═════════════════════════")
	fmt.Println()

	// Raw data stats
	fmt.Printf("Raw Data (Minute-level):\n")
	fmt.Printf("  Records:     %10d\n", stats.RawRecords)
	if stats.OldestRaw != "" {
		fmt.Printf("  Oldest date: %s\n", stats.OldestRaw)
		oldestDate, _ := time.Parse("2006-01-02", stats.OldestRaw)
		daysOld := int(time.Since(oldestDate).Hours() / 24)
		fmt.Printf("  Data age:    %d days\n", daysOld)
	}
	fmt.Println()

	// Hourly data stats
	fmt.Printf("Hourly Aggregates:\n")
	fmt.Printf("  Records:     %10d\n", stats.HourlyRecords)
	if stats.OldestHourly != "" {
		fmt.Printf("  Oldest date: %s\n", stats.OldestHourly)
		oldestDate, _ := time.Parse("2006-01-02", stats.OldestHourly)
		daysOld := int(time.Since(oldestDate).Hours() / 24)
		fmt.Printf("  Data age:    %d days\n", daysOld)
	}
	fmt.Println()

	// Daily data stats
	fmt.Printf("Daily Aggregates:\n")
	fmt.Printf("  Records:     %10d\n", stats.DailyRecords)
	if stats.OldestDaily != "" {
		fmt.Printf("  Oldest date: %s\n", stats.OldestDaily)
		oldestDate, _ := time.Parse("2006-01-02", stats.OldestDaily)
		daysOld := int(time.Since(oldestDate).Hours() / 24)
		fmt.Printf("  Data age:    %d days\n", daysOld)
	}
	fmt.Println()

	// Total and recommendations
	totalRecords := stats.RawRecords + stats.HourlyRecords + stats.DailyRecords
	fmt.Printf("Total Records: %d\n", totalRecords)
	fmt.Println()

	// Estimate storage savings
	if stats.RawRecords > 0 {
		// Rough estimate: raw record ~1KB, hourly ~100bytes, daily ~100bytes
		estimatedSize := (stats.RawRecords * 1024) + (stats.HourlyRecords * 100) + (stats.DailyRecords * 100)
		fmt.Printf("Estimated Size: ~%d MB\n", estimatedSize/(1024*1024))

		// What it would be without retention
		daysOfData := 0
		if stats.OldestRaw != "" {
			oldestDate, _ := time.Parse("2006-01-02", stats.OldestRaw)
			daysOfData = int(time.Since(oldestDate).Hours() / 24)
		}
		if daysOfData > 90 {
			withoutRetention := int64(daysOfData) * (stats.RawRecords / int64(daysOfData)) * 1024
			savings := withoutRetention - estimatedSize
			savingsPercent := float64(savings) / float64(withoutRetention) * 100
			fmt.Printf("Storage Saved:  ~%d MB (%.1f%% reduction)\n", savings/(1024*1024), savingsPercent)
		}
	}

	return nil
}
