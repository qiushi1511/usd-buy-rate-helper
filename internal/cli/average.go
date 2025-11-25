package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
	"github.com/qiushi1511/usd-buy-rate-monitor/pkg/chart"
)

// AverageCommand handles the average command functionality
type AverageCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewAverageCommand creates a new average command handler
func NewAverageCommand(repo *storage.Repository, logger *slog.Logger) *AverageCommand {
	return &AverageCommand{
		repo:   repo,
		logger: logger,
	}
}

// DisplayAverage shows the daily average exchange rate
func (a *AverageCommand) DisplayAverage(ctx context.Context, dates []string, compare bool, showChart bool) error {
	if len(dates) == 0 {
		return fmt.Errorf("no dates specified")
	}

	var allStats []*storage.DailyStats

	fmt.Printf("\n")
	fmt.Printf("Daily Average Exchange Rates\n")
	fmt.Printf("════════════════════════════\n")
	fmt.Printf("\n")

	for _, date := range dates {
		stats, err := a.repo.GetDailyStats(ctx, date)
		if err != nil {
			return fmt.Errorf("getting stats for %s: %w", date, err)
		}

		if stats.SampleCount == 0 {
			fmt.Printf("%-12s  No data available\n\n", date)
			continue
		}

		allStats = append(allStats, stats)

		fmt.Printf("%-12s\n", date)
		fmt.Printf("  Average:     %.4f CNY\n", stats.AvgRate)
		fmt.Printf("  Min:         %.4f CNY\n", stats.MinRate)
		fmt.Printf("  Max:         %.4f CNY\n", stats.MaxRate)
		fmt.Printf("  Peak Time:   %s\n", stats.PeakTime.Format("15:04:05"))
		fmt.Printf("  Samples:     %d\n", stats.SampleCount)
		fmt.Printf("  Volatility:  %.4f CNY\n", stats.MaxRate-stats.MinRate)
		fmt.Printf("\n")
	}

	// Display comparison if requested and we have multiple dates
	if compare && len(allStats) > 1 {
		a.displayComparison(allStats)
	}

	// Display charts if requested and we have data
	if showChart && len(allStats) > 0 {
		a.displayCharts(allStats)
	}

	return nil
}

func (a *AverageCommand) displayCharts(stats []*storage.DailyStats) {
	width, height := chart.GetTerminalDimensions()

	// Average rate trend chart
	fmt.Printf("\n")
	fmt.Println(chart.RenderDailyAverageChart(stats, width, height))
	fmt.Printf("\n")

	// Volatility chart
	if len(stats) > 1 {
		fmt.Println(chart.RenderVolatilityChart(stats, width, height))
		fmt.Printf("\n")
	}
}

// DisplayAverageRange shows average rates for a range of recent days
func (a *AverageCommand) DisplayAverageRange(ctx context.Context, days int, compare bool, showChart bool) error {
	dates := getRecentDates(days)

	var allStats []*storage.DailyStats

	fmt.Printf("\n")
	fmt.Printf("Daily Average Exchange Rates (Last %d Days)\n", days)
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("\n")

	// Display table header
	fmt.Printf("%-12s  %-10s  %-10s  %-10s  %-10s  %-8s\n",
		"Date", "Average", "Min", "Max", "Volatility", "Samples")
	fmt.Printf("%s\n", strings.Repeat("─", 75))

	for _, date := range dates {
		stats, err := a.repo.GetDailyStats(ctx, date)
		if err != nil {
			a.logger.Warn("failed to get stats", "date", date, "error", err)
			continue
		}

		if stats.SampleCount == 0 {
			fmt.Printf("%-12s  %-10s  %-10s  %-10s  %-10s  %-8s\n",
				date, "No data", "-", "-", "-", "0")
			continue
		}

		allStats = append(allStats, stats)

		fmt.Printf("%-12s  %10.4f  %10.4f  %10.4f  %10.4f  %8d\n",
			date,
			stats.AvgRate,
			stats.MinRate,
			stats.MaxRate,
			stats.MaxRate-stats.MinRate,
			stats.SampleCount)
	}

	fmt.Printf("\n")

	// Display comparison if requested and we have data
	if compare && len(allStats) > 0 {
		a.displayComparison(allStats)
	}

	// Display charts if requested and we have data
	if showChart && len(allStats) > 0 {
		a.displayCharts(allStats)
	}

	return nil
}

func (a *AverageCommand) displayComparison(allStats []*storage.DailyStats) {
	if len(allStats) == 0 {
		return
	}

	fmt.Printf("Comparison Across Dates\n")
	fmt.Printf("══════════════════════\n")
	fmt.Printf("\n")

	// Calculate overall statistics
	var totalAvg, totalMin, totalMax float64
	totalMin = allStats[0].MinRate
	totalMax = allStats[0].MaxRate

	var avgSum float64
	for _, stats := range allStats {
		avgSum += stats.AvgRate
		if stats.MinRate < totalMin {
			totalMin = stats.MinRate
		}
		if stats.MaxRate > totalMax {
			totalMax = stats.MaxRate
		}
	}
	totalAvg = avgSum / float64(len(allStats))

	fmt.Printf("  Overall Average:    %.4f CNY\n", totalAvg)
	fmt.Printf("  Absolute Minimum:   %.4f CNY\n", totalMin)
	fmt.Printf("  Absolute Maximum:   %.4f CNY\n", totalMax)
	fmt.Printf("  Total Range:        %.4f CNY\n", totalMax-totalMin)
	fmt.Printf("\n")

	// Day-to-day changes
	if len(allStats) > 1 {
		fmt.Printf("Day-to-Day Changes:\n")
		for i := len(allStats) - 1; i > 0; i-- {
			prev := allStats[i]
			curr := allStats[i-1]

			delta := curr.AvgRate - prev.AvgRate
			deltaPercent := (delta / prev.AvgRate) * 100

			symbol := "→"
			trend := "stable"
			if delta > 0 {
				symbol = "↑"
				trend = "up"
			} else if delta < 0 {
				symbol = "↓"
				trend = "down"
			}

			fmt.Printf("  %s → %s:  %s %.4f (%.2f%%) [%s]\n",
				prev.Date,
				curr.Date,
				symbol,
				delta,
				deltaPercent,
				trend)
		}
		fmt.Printf("\n")
	}

	// Volatility analysis
	var totalVolatility float64
	for _, stats := range allStats {
		totalVolatility += (stats.MaxRate - stats.MinRate)
	}
	avgVolatility := totalVolatility / float64(len(allStats))

	fmt.Printf("Volatility Analysis:\n")
	fmt.Printf("  Average Daily Range:  %.4f CNY\n", avgVolatility)

	// Find most and least volatile days
	var mostVolatile, leastVolatile *storage.DailyStats
	for _, stats := range allStats {
		volatility := stats.MaxRate - stats.MinRate
		if mostVolatile == nil || volatility > (mostVolatile.MaxRate-mostVolatile.MinRate) {
			mostVolatile = stats
		}
		if leastVolatile == nil || volatility < (leastVolatile.MaxRate-leastVolatile.MinRate) {
			leastVolatile = stats
		}
	}

	if mostVolatile != nil {
		fmt.Printf("  Most Volatile Day:    %s (%.4f CNY range)\n",
			mostVolatile.Date,
			mostVolatile.MaxRate-mostVolatile.MinRate)
	}
	if leastVolatile != nil {
		fmt.Printf("  Least Volatile Day:   %s (%.4f CNY range)\n",
			leastVolatile.Date,
			leastVolatile.MaxRate-leastVolatile.MinRate)
	}
	fmt.Printf("\n")
}
