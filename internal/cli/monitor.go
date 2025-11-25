package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// MonitorCommand handles the monitor command functionality
type MonitorCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewMonitorCommand creates a new monitor command handler
func NewMonitorCommand(repo *storage.Repository, logger *slog.Logger) *MonitorCommand {
	return &MonitorCommand{
		repo:   repo,
		logger: logger,
	}
}

// DisplayCurrent shows the current/latest exchange rate
func (m *MonitorCommand) DisplayCurrent(ctx context.Context) error {
	rate, err := m.repo.GetLatestRate(ctx)
	if err != nil {
		return fmt.Errorf("getting latest rate: %w", err)
	}

	if rate == nil {
		fmt.Println("No data available yet. Make sure the daemon is running.")
		return nil
	}

	fmt.Printf("\n")
	fmt.Printf("USD/CNY Exchange Rate\n")
	fmt.Printf("═════════════════════\n")
	fmt.Printf("\n")
	fmt.Printf("  Rate:      %.4f CNY\n", rate.RtcBid)
	fmt.Printf("  Time:      %s\n", rate.CollectedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Age:       %s ago\n", formatDuration(time.Since(rate.CollectedAt)))
	fmt.Printf("\n")

	// Get previous rate for comparison
	prevRate, err := m.getPreviousRate(ctx, rate.CollectedAt)
	if err == nil && prevRate != nil {
		delta := rate.RtcBid - prevRate.RtcBid
		deltaPercent := (delta / prevRate.RtcBid) * 100

		symbol := "→"
		if delta > 0 {
			symbol = "↑"
		} else if delta < 0 {
			symbol = "↓"
		}

		fmt.Printf("  Change:    %s %.4f (%.2f%%)\n", symbol, delta, deltaPercent)
		fmt.Printf("  Previous:  %.4f CNY at %s\n",
			prevRate.RtcBid,
			prevRate.CollectedAt.Format("15:04:05"))
		fmt.Printf("\n")
	}

	return nil
}

// DisplayRealtime shows real-time updates with the specified refresh interval
func (m *MonitorCommand) DisplayRealtime(ctx context.Context, refresh time.Duration) error {
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	// Clear screen and show initial data
	clearScreen()
	if err := m.displayRealtimeOnce(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			clearScreen()
			if err := m.displayRealtimeOnce(ctx); err != nil {
				m.logger.Warn("failed to refresh display", "error", err)
			}
		}
	}
}

func (m *MonitorCommand) displayRealtimeOnce(ctx context.Context) error {
	rate, err := m.repo.GetLatestRate(ctx)
	if err != nil {
		return fmt.Errorf("getting latest rate: %w", err)
	}

	if rate == nil {
		fmt.Println("\nNo data available yet. Make sure the daemon is running.\n")
		return nil
	}

	now := time.Now()
	fmt.Printf("USD/CNY Exchange Rate Monitor\n")
	fmt.Printf("═════════════════════════════\n")
	fmt.Printf("\n")
	fmt.Printf("  Current Rate:    %.4f CNY\n", rate.RtcBid)
	fmt.Printf("  Last Updated:    %s (%s ago)\n",
		rate.CollectedAt.Format("2006-01-02 15:04:05"),
		formatDuration(now.Sub(rate.CollectedAt)))
	fmt.Printf("\n")

	// Get previous rate for comparison
	prevRate, err := m.getPreviousRate(ctx, rate.CollectedAt)
	if err == nil && prevRate != nil {
		delta := rate.RtcBid - prevRate.RtcBid
		deltaPercent := (delta / prevRate.RtcBid) * 100

		symbol := "→"
		color := ""
		if delta > 0 {
			symbol = "↑"
			color = " (increasing)"
		} else if delta < 0 {
			symbol = "↓"
			color = " (decreasing)"
		}

		fmt.Printf("  Change:          %s %.4f (%.2f%%)%s\n", symbol, delta, deltaPercent, color)
		fmt.Printf("  Previous Rate:   %.4f CNY\n", prevRate.RtcBid)
		fmt.Printf("\n")
	}

	// Show total records
	count, err := m.repo.Count(ctx)
	if err == nil {
		fmt.Printf("  Total Records:   %d\n", count)
	}

	fmt.Printf("\n")
	fmt.Printf("  Press Ctrl+C to exit\n")
	fmt.Printf("  Refreshed at: %s\n", now.Format("15:04:05"))

	return nil
}

func (m *MonitorCommand) getPreviousRate(ctx context.Context, currentTime time.Time) (*storage.ExchangeRate, error) {
	// Get rates from the last 2 hours to find the previous reading
	start := currentTime.Add(-2 * time.Hour)
	end := currentTime.Add(-1 * time.Second) // Exclude current reading

	rates, err := m.repo.GetRatesByTimeRange(ctx, start, end)
	if err != nil {
		return nil, err
	}

	if len(rates) == 0 {
		return nil, nil
	}

	// Return the most recent rate (last in the list since ordered by ASC)
	return &rates[len(rates)-1], nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

func clearScreen() {
	// ANSI escape code to clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")
}
