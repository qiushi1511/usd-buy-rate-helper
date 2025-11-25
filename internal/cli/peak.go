package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// PeakCommand handles the peak command functionality
type PeakCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewPeakCommand creates a new peak command handler
func NewPeakCommand(repo *storage.Repository, logger *slog.Logger) *PeakCommand {
	return &PeakCommand{
		repo:   repo,
		logger: logger,
	}
}

// DisplayPeak shows the daily peak exchange rate
func (p *PeakCommand) DisplayPeak(ctx context.Context, dates []string) error {
	if len(dates) == 0 {
		return fmt.Errorf("no dates specified")
	}

	fmt.Printf("\n")
	fmt.Printf("Daily Peak Exchange Rates\n")
	fmt.Printf("═════════════════════════\n")
	fmt.Printf("\n")

	for _, date := range dates {
		peak, err := p.repo.GetDailyPeak(ctx, date)
		if err != nil {
			return fmt.Errorf("getting peak for %s: %w", date, err)
		}

		if peak == nil {
			fmt.Printf("%-12s  No data available\n", date)
			continue
		}

		fmt.Printf("%-12s\n", date)
		fmt.Printf("  Peak Rate:  %.4f CNY\n", peak.RtcBid)
		fmt.Printf("  Time:       %s\n", peak.CollectedAt.Format("15:04:05"))
		fmt.Printf("\n")
	}

	return nil
}

// DisplayPeakRange shows peak rates for a range of recent days
func (p *PeakCommand) DisplayPeakRange(ctx context.Context, days int) error {
	dates := getRecentDates(days)

	fmt.Printf("\n")
	fmt.Printf("Daily Peak Exchange Rates (Last %d Days)\n", days)
	fmt.Printf("═════════════════════════════════════════\n")
	fmt.Printf("\n")

	// Display table header
	fmt.Printf("%-12s  %-10s  %-10s\n", "Date", "Peak (CNY)", "Time")
	fmt.Printf("%s\n", strings.Repeat("─", 40))

	var peaks []struct {
		date string
		rate *storage.ExchangeRate
	}

	for _, date := range dates {
		peak, err := p.repo.GetDailyPeak(ctx, date)
		if err != nil {
			p.logger.Warn("failed to get peak", "date", date, "error", err)
			continue
		}

		peaks = append(peaks, struct {
			date string
			rate *storage.ExchangeRate
		}{date: date, rate: peak})

		if peak == nil {
			fmt.Printf("%-12s  %-10s  %-10s\n", date, "No data", "-")
		} else {
			fmt.Printf("%-12s  %10.4f  %-10s\n",
				date,
				peak.RtcBid,
				peak.CollectedAt.Format("15:04:05"))
		}
	}

	fmt.Printf("\n")

	// Display summary statistics if we have data
	var validPeaks []float64
	for _, p := range peaks {
		if p.rate != nil {
			validPeaks = append(validPeaks, p.rate.RtcBid)
		}
	}

	if len(validPeaks) > 0 {
		var maxPeak, minPeak, sumPeak float64
		maxPeak = validPeaks[0]
		minPeak = validPeaks[0]

		for _, rate := range validPeaks {
			if rate > maxPeak {
				maxPeak = rate
			}
			if rate < minPeak {
				minPeak = rate
			}
			sumPeak += rate
		}
		avgPeak := sumPeak / float64(len(validPeaks))

		fmt.Printf("Summary:\n")
		fmt.Printf("  Highest Peak:   %.4f CNY\n", maxPeak)
		fmt.Printf("  Lowest Peak:    %.4f CNY\n", minPeak)
		fmt.Printf("  Average Peak:   %.4f CNY\n", avgPeak)
		fmt.Printf("  Peak Range:     %.4f CNY\n", maxPeak-minPeak)
		fmt.Printf("\n")
	}

	return nil
}

func getRecentDates(days int) []string {
	dates := make([]string, days)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		dates[i] = date.Format("2006-01-02")
	}

	return dates
}
