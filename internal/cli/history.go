package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
	"github.com/qiushi1511/usd-buy-rate-monitor/pkg/chart"
)

// HistoryCommand handles the history command functionality
type HistoryCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewHistoryCommand creates a new history command handler
func NewHistoryCommand(repo *storage.Repository, logger *slog.Logger) *HistoryCommand {
	return &HistoryCommand{
		repo:   repo,
		logger: logger,
	}
}

// DisplayHistory shows exchange rates for a specific time range
func (h *HistoryCommand) DisplayHistory(ctx context.Context, start, end time.Time, format string, showChart bool) error {
	rates, err := h.repo.GetRatesByTimeRange(ctx, start, end)
	if err != nil {
		return fmt.Errorf("querying rates: %w", err)
	}

	if len(rates) == 0 {
		fmt.Printf("No data found for the time range %s to %s\n",
			start.Format("2006-01-02 15:04:05"),
			end.Format("2006-01-02 15:04:05"))
		return nil
	}

	switch format {
	case "table":
		h.displayTable(rates, start, end)
	case "csv":
		h.displayCSV(rates)
	case "json":
		h.displayJSON(rates)
	case "chart":
		h.displayChart(rates)
		return nil
	default:
		h.displayTable(rates, start, end)
	}

	// Show chart after table if requested
	if showChart && format != "chart" {
		h.displayChart(rates)
	}

	return nil
}

func (h *HistoryCommand) displayChart(rates []storage.ExchangeRate) {
	width, height := chart.GetTerminalDimensions()
	chart.PrintChartWithStats(rates, width, height)
}

func (h *HistoryCommand) displayTable(rates []storage.ExchangeRate, start, end time.Time) {
	fmt.Printf("\n")
	fmt.Printf("Exchange Rate History\n")
	fmt.Printf("═════════════════════\n")
	fmt.Printf("Period: %s to %s\n",
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"))
	fmt.Printf("Records: %d\n", len(rates))
	fmt.Printf("\n")

	// Calculate statistics
	var minRate, maxRate, sumRate float64
	minRate = rates[0].RtcBid
	maxRate = rates[0].RtcBid

	for _, rate := range rates {
		if rate.RtcBid < minRate {
			minRate = rate.RtcBid
		}
		if rate.RtcBid > maxRate {
			maxRate = rate.RtcBid
		}
		sumRate += rate.RtcBid
	}
	avgRate := sumRate / float64(len(rates))

	fmt.Printf("Summary Statistics:\n")
	fmt.Printf("  Min:     %.4f CNY\n", minRate)
	fmt.Printf("  Max:     %.4f CNY\n", maxRate)
	fmt.Printf("  Average: %.4f CNY\n", avgRate)
	fmt.Printf("  Range:   %.4f CNY\n", maxRate-minRate)
	fmt.Printf("\n")

	// Display table header
	fmt.Printf("%-20s  %-10s  %-8s\n", "Time", "Rate (CNY)", "Change")
	fmt.Printf("%s\n", strings.Repeat("─", 50))

	var prevRate *float64
	for _, rate := range rates {
		changeStr := "   -    "
		if prevRate != nil {
			delta := rate.RtcBid - *prevRate
			symbol := " "
			if delta > 0 {
				symbol = "↑"
			} else if delta < 0 {
				symbol = "↓"
			} else {
				symbol = "→"
			}
			changeStr = fmt.Sprintf("%s%+.4f", symbol, delta)
		}

		fmt.Printf("%-20s  %10.4f  %-8s\n",
			rate.CollectedAt.Format("2006-01-02 15:04:05"),
			rate.RtcBid,
			changeStr)

		prevRate = &rate.RtcBid
	}
	fmt.Printf("\n")
}

func (h *HistoryCommand) displayCSV(rates []storage.ExchangeRate) {
	fmt.Printf("Timestamp,Rate,Date,Time\n")
	for _, rate := range rates {
		fmt.Printf("%s,%.4f,%s,%s\n",
			rate.CollectedAt.Format("2006-01-02 15:04:05"),
			rate.RtcBid,
			rate.CollectedAt.Format("2006-01-02"),
			rate.CollectedAt.Format("15:04:05"))
	}
}

func (h *HistoryCommand) displayJSON(rates []storage.ExchangeRate) {
	fmt.Printf("[\n")
	for i, rate := range rates {
		comma := ","
		if i == len(rates)-1 {
			comma = ""
		}
		fmt.Printf("  {\n")
		fmt.Printf("    \"timestamp\": \"%s\",\n", rate.CollectedAt.Format(time.RFC3339))
		fmt.Printf("    \"rate\": %.4f,\n", rate.RtcBid)
		fmt.Printf("    \"currency\": \"%s\"\n", rate.CurrencyCode)
		fmt.Printf("  }%s\n", comma)
	}
	fmt.Printf("]\n")
}

// ParseTimeRange parses start and end times from various formats
func ParseTimeRange(startStr, endStr, lastDuration string) (time.Time, time.Time, error) {
	now := time.Now()

	// If "last" duration is specified (e.g., "2h", "30m")
	if lastDuration != "" {
		duration, err := time.ParseDuration(lastDuration)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %w", err)
		}
		return now.Add(-duration), now, nil
	}

	// Parse start and end times
	var start, end time.Time
	var err error

	if startStr == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("start time is required (or use --last)")
	}

	// Try different time formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"15:04:05",
		"15:04",
	}

	for _, format := range formats {
		start, err = time.ParseInLocation(format, startStr, time.Local)
		if err == nil {
			// If only time is provided (no date), use today's date
			if format == "15:04:05" || format == "15:04" {
				start = time.Date(now.Year(), now.Month(), now.Day(),
					start.Hour(), start.Minute(), start.Second(), 0, time.Local)
			}
			break
		}
	}
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start time format: %s", startStr)
	}

	// If end time is not specified, use current time
	if endStr == "" {
		end = now
	} else {
		for _, format := range formats {
			end, err = time.ParseInLocation(format, endStr, time.Local)
			if err == nil {
				// If only time is provided (no date), use today's date
				if format == "15:04:05" || format == "15:04" {
					end = time.Date(now.Year(), now.Month(), now.Day(),
						end.Hour(), end.Minute(), end.Second(), 0, time.Local)
				}
				break
			}
		}
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end time format: %s", endStr)
		}
	}

	// Validate that start is before end
	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start time must be before end time")
	}

	return start, end, nil
}
