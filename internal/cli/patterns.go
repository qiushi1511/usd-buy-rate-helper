package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// PatternsCommand handles the patterns command functionality
type PatternsCommand struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewPatternsCommand creates a new patterns command handler
func NewPatternsCommand(repo *storage.Repository, logger *slog.Logger) *PatternsCommand {
	return &PatternsCommand{
		repo:   repo,
		logger: logger,
	}
}

// DisplayPatterns shows historical patterns in exchange rates
func (p *PatternsCommand) DisplayPatterns(ctx context.Context, days, weeks int) error {
	fmt.Printf("\n")
	fmt.Printf("Exchange Rate Patterns Analysis\n")
	fmt.Printf("═══════════════════════════════\n")
	fmt.Printf("Analyzing last %d days of data\n", days)
	fmt.Printf("\n")

	// Get hourly patterns
	hourlyPatterns, err := p.repo.GetHourlyPatterns(ctx, days)
	if err != nil {
		return fmt.Errorf("getting hourly patterns: %w", err)
	}

	if len(hourlyPatterns) == 0 {
		fmt.Println("No data available for pattern analysis")
		return nil
	}

	// Display hourly patterns
	p.displayHourlyPatterns(hourlyPatterns, days)

	// Get day of week patterns if we have enough data
	if weeks > 0 {
		dowPatterns, err := p.repo.GetDayOfWeekPatterns(ctx, weeks)
		if err != nil {
			p.logger.Warn("failed to get day of week patterns", "error", err)
		} else if len(dowPatterns) > 0 {
			p.displayDayOfWeekPatterns(dowPatterns, weeks)
		}
	}

	return nil
}

func (p *PatternsCommand) displayHourlyPatterns(patterns []storage.HourlyPattern, days int) {
	fmt.Printf("Hourly Patterns (Business Hours 08:00-22:00 CST)\n")
	fmt.Printf("─────────────────────────────────────────────────\n")
	fmt.Printf("\n")

	// Table header
	fmt.Printf("%-8s  %-10s  %-10s  %-10s  %-8s  %-12s\n",
		"Hour", "Avg Rate", "Min", "Max", "Samples", "Peak Freq")
	fmt.Printf("%s\n", strings.Repeat("─", 75))

	// Find hour with highest average and most peaks
	var highestAvgHour, mostPeaksHour *storage.HourlyPattern
	for i := range patterns {
		if highestAvgHour == nil || patterns[i].AvgRate > highestAvgHour.AvgRate {
			highestAvgHour = &patterns[i]
		}
		if mostPeaksHour == nil || patterns[i].PeakFreq > mostPeaksHour.PeakFreq {
			mostPeaksHour = &patterns[i]
		}
	}

	// Display patterns
	for _, pattern := range patterns {
		peakIndicator := ""
		if highestAvgHour != nil && pattern.Hour == highestAvgHour.Hour {
			peakIndicator = " ⭐ Highest avg"
		} else if mostPeaksHour != nil && pattern.Hour == mostPeaksHour.Hour && pattern.PeakFreq > 0 {
			peakIndicator = " 🏆 Most peaks"
		}

		peakFreqPct := 0.0
		if days > 0 {
			peakFreqPct = (float64(pattern.PeakFreq) / float64(days)) * 100
		}

		fmt.Printf("%02d:00    %10.4f  %10.4f  %10.4f  %8d  %3d (%4.1f%%)%s\n",
			pattern.Hour,
			pattern.AvgRate,
			pattern.MinRate,
			pattern.MaxRate,
			pattern.SampleCount,
			pattern.PeakFreq,
			peakFreqPct,
			peakIndicator,
		)
	}
	fmt.Printf("\n")

	// Key insights
	fmt.Printf("Key Insights:\n")
	if highestAvgHour != nil {
		fmt.Printf("  • Highest average rate: %02d:00 (%.4f CNY)\n",
			highestAvgHour.Hour, highestAvgHour.AvgRate)
	}
	if mostPeaksHour != nil && mostPeaksHour.PeakFreq > 0 {
		peakPct := (float64(mostPeaksHour.PeakFreq) / float64(days)) * 100
		fmt.Printf("  • Peak time: %02d:00 (%d/%d days = %.1f%%)\n",
			mostPeaksHour.Hour, mostPeaksHour.PeakFreq, days, peakPct)
	}

	// Find volatility window (highest range)
	var maxRange float64
	var maxRangeHour int
	for _, p := range patterns {
		rng := p.MaxRate - p.MinRate
		if rng > maxRange {
			maxRange = rng
			maxRangeHour = p.Hour
		}
	}
	if maxRange > 0 {
		fmt.Printf("  • Most volatile hour: %02d:00 (range: %.4f CNY)\n",
			maxRangeHour, maxRange)
	}
	fmt.Printf("\n")
}

func (p *PatternsCommand) displayDayOfWeekPatterns(patterns []storage.DayOfWeekPattern, weeks int) {
	fmt.Printf("Day of Week Patterns (Last %d weeks)\n", weeks)
	fmt.Printf("────────────────────────────────────\n")
	fmt.Printf("\n")

	// Table header
	fmt.Printf("%-10s  %-10s  %-10s  %-10s  %-12s  %-8s\n",
		"Day", "Avg Rate", "Min", "Max", "Avg Range", "Days")
	fmt.Printf("%s\n", strings.Repeat("─", 75))

	// Find best and worst days
	var bestDay, worstDay *storage.DayOfWeekPattern
	for i := range patterns {
		if bestDay == nil || patterns[i].AvgRate > bestDay.AvgRate {
			bestDay = &patterns[i]
		}
		if worstDay == nil || patterns[i].AvgRate < worstDay.AvgRate {
			worstDay = &patterns[i]
		}
	}

	// Display patterns
	for _, pattern := range patterns {
		indicator := ""
		if bestDay != nil && pattern.DayOfWeek == bestDay.DayOfWeek {
			indicator = " ⭐ Best"
		} else if worstDay != nil && pattern.DayOfWeek == worstDay.DayOfWeek {
			indicator = " ↓ Lowest"
		}

		fmt.Printf("%-10s  %10.4f  %10.4f  %10.4f  %10.4f    %8d%s\n",
			pattern.DayName,
			pattern.AvgRate,
			pattern.MinRate,
			pattern.MaxRate,
			pattern.AvgRange,
			pattern.SampleDays,
			indicator,
		)
	}
	fmt.Printf("\n")

	// Key insights
	fmt.Printf("Weekly Insights:\n")
	if bestDay != nil && worstDay != nil {
		diff := bestDay.AvgRate - worstDay.AvgRate
		fmt.Printf("  • Best day: %s (avg %.4f CNY)\n",
			bestDay.DayName, bestDay.AvgRate)
		fmt.Printf("  • Lowest day: %s (avg %.4f CNY)\n",
			worstDay.DayName, worstDay.AvgRate)
		fmt.Printf("  • Weekly variance: %.4f CNY\n", diff)
	}

	// Find most volatile day
	var mostVolatile *storage.DayOfWeekPattern
	for i := range patterns {
		if mostVolatile == nil || patterns[i].AvgRange > mostVolatile.AvgRange {
			mostVolatile = &patterns[i]
		}
	}
	if mostVolatile != nil {
		fmt.Printf("  • Most volatile day: %s (avg range: %.4f CNY)\n",
			mostVolatile.DayName, mostVolatile.AvgRange)
	}
	fmt.Printf("\n")
}
