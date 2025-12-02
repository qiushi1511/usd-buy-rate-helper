package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/recommender"
	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// RecommendCommand handles the recommend command functionality
type RecommendCommand struct {
	repo       *storage.Repository
	recommender *recommender.Recommender
	logger     *slog.Logger
}

// NewRecommendCommand creates a new recommend command handler
func NewRecommendCommand(repo *storage.Repository, logger *slog.Logger) *RecommendCommand {
	return &RecommendCommand{
		repo:       repo,
		recommender: recommender.NewRecommender(repo, logger),
		logger:     logger,
	}
}

// DisplayRecommendation shows the exchange recommendation
func (c *RecommendCommand) DisplayRecommendation(ctx context.Context, amount float64, showDetails bool) error {
	rec, err := c.recommender.GetRecommendation(ctx, amount)
	if err != nil {
		return fmt.Errorf("getting recommendation: %w", err)
	}

	// Print header
	fmt.Printf("\n")
	fmt.Printf("Exchange Recommendation\n")
	fmt.Printf("═══════════════════════\n")
	fmt.Printf("\n")

	// Current rate and conversion
	fmt.Printf("Current Rate:    %.4f CNY per USD\n", rec.CurrentRate)
	if amount > 0 {
		fmt.Printf("Amount:          %s RMB → %s USD\n",
			formatMoney(rec.Amount),
			formatMoney(rec.USDAmount))
	}
	fmt.Printf("\n")

	// Main recommendation
	c.displayActionRecommendation(rec)

	// Confidence
	fmt.Printf("Confidence:      %.0f%% (%s)\n", rec.ConfidenceScore, rec.Confidence)
	fmt.Printf("\n")

	// Analysis reasoning
	fmt.Printf("Analysis:\n")
	for _, reason := range rec.Reasoning {
		fmt.Printf("  • %s\n", reason)
	}
	fmt.Printf("\n")

	// Show percentile context
	ranking, _ := c.recommender.GetHistoricalRanking(ctx, rec.CurrentRate, 30)
	fmt.Printf("Historical Context (Last 30 Days):\n")
	fmt.Printf("  Percentile:      %.0fth (out of 100)\n", rec.PercentileRank)
	fmt.Printf("  Ranking:         %s\n", ranking)
	fmt.Printf("  30-Day Average:  %.4f CNY\n", rec.HistoricalStats.AvgRate30Days)
	fmt.Printf("  30-Day Range:    %.4f - %.4f CNY\n", rec.HistoricalStats.MinRate30Days, rec.HistoricalStats.MaxRate30Days)
	fmt.Printf("\n")

	// Potential gain/loss
	if amount > 0 && (rec.PotentialGain > 0 || rec.PotentialLoss > 0) {
		fmt.Printf("Risk/Reward Assessment:\n")
		if rec.PotentialGain > 0 {
			fmt.Printf("  Potential Gain:  +%s USD (if optimal timing)\n", formatMoney(rec.PotentialGain))
		}
		if rec.PotentialLoss > 0 {
			fmt.Printf("  Downside Risk:   -%s USD (worst case scenario)\n", formatMoney(rec.PotentialLoss))
		}
		fmt.Printf("  Risk Level:      %s\n", rec.RiskLevel)
		fmt.Printf("\n")
	}

	// Optimal window if available
	if rec.OptimalWindow != nil && rec.Action == recommender.ActionWait {
		c.displayOptimalWindow(rec.OptimalWindow)
	}

	// Next check time
	fmt.Printf("Next Check:      %s\n", formatTimeFromNow(rec.NextCheckTime))
	fmt.Printf("\n")

	// Show detailed predictions if requested
	if showDetails && len(rec.PredictedNextHours) > 0 {
		c.displayPredictions(rec.PredictedNextHours, rec.CurrentRate)
	}

	return nil
}

// displayActionRecommendation shows the main recommendation with visual emphasis
func (c *RecommendCommand) displayActionRecommendation(rec *recommender.Recommendation) {
	switch rec.Action {
	case recommender.ActionExchangeNow:
		fmt.Printf("Recommendation:  🟢 EXCHANGE NOW\n")
		fmt.Printf("\n")
		fmt.Printf("The current rate is favorable. Consider exchanging immediately.\n")
	case recommender.ActionWait:
		fmt.Printf("Recommendation:  ⏳ WAIT - Better Rate Expected\n")
		fmt.Printf("\n")
		fmt.Printf("Historical patterns suggest waiting for a better rate.\n")
	case recommender.ActionNeutral:
		fmt.Printf("Recommendation:  ⚪ NEUTRAL - Your Choice\n")
		fmt.Printf("\n")
		fmt.Printf("Current rate is acceptable but not exceptional. Exchange now or wait based on urgency.\n")
	}
	fmt.Printf("\n")
}

// displayOptimalWindow shows the predicted optimal exchange window
func (c *RecommendCommand) displayOptimalWindow(window *recommender.TimeWindow) {
	fmt.Printf("Optimal Exchange Window:\n")
	fmt.Printf("  Time:            %02d:00 - %02d:00 CST\n", window.StartHour, window.EndHour)

	now := time.Now()
	if window.StartTime.After(now) {
		duration := window.StartTime.Sub(now)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours > 0 {
			fmt.Printf("  Starts in:       %d hours %d minutes\n", hours, minutes)
		} else {
			fmt.Printf("  Starts in:       %d minutes\n", minutes)
		}
	} else if window.EndTime.After(now) {
		fmt.Printf("  Status:          ✅ Active NOW!\n")
	}

	fmt.Printf("  Expected Rate:   %.4f CNY (+%.2f%%)\n",
		window.ExpectedRate,
		((window.ExpectedRate / window.ExpectedRate) - 1) * 100)
	fmt.Printf("  Confidence:      %.0f%%\n", window.Probability)
	fmt.Printf("  Reasoning:       %s\n", window.ReasoningText)
	fmt.Printf("\n")
}

// displayPredictions shows rate predictions for upcoming hours
func (c *RecommendCommand) displayPredictions(predictions []recommender.HourPrediction, currentRate float64) {
	fmt.Printf("Rate Predictions (Next Few Hours):\n")
	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("%-8s  %-12s  %-10s  %-10s\n", "Time", "Predicted", "Change", "Confidence")
	fmt.Printf("─────────────────────────────────────────────────────────────\n")

	for _, pred := range predictions {
		change := pred.PredictedRate - currentRate
		changePct := (change / currentRate) * 100
		changeSymbol := "→"
		if change > 0 {
			changeSymbol = "↑"
		} else if change < 0 {
			changeSymbol = "↓"
		}

		fmt.Printf("%-8s  %.4f CNY  %s %+.4f  %.0f%%\n",
			pred.TimeLabel,
			pred.PredictedRate,
			changeSymbol,
			changePct,
			pred.Confidence,
		)
	}
	fmt.Printf("\n")
	fmt.Printf("Note: Predictions based on historical patterns. Actual rates may vary.\n")
	fmt.Printf("\n")
}

// DisplayQuickCheck shows a simplified one-line recommendation
func (c *RecommendCommand) DisplayQuickCheck(ctx context.Context) error {
	rec, err := c.recommender.GetRecommendation(ctx, 0)
	if err != nil {
		return fmt.Errorf("getting recommendation: %w", err)
	}

	var actionText string
	switch rec.Action {
	case recommender.ActionExchangeNow:
		actionText = "🟢 EXCHANGE NOW"
	case recommender.ActionWait:
		actionText = "⏳ WAIT"
	case recommender.ActionNeutral:
		actionText = "⚪ NEUTRAL"
	}

	fmt.Printf("\n")
	fmt.Printf("Rate: %.4f CNY  |  %s  |  Confidence: %.0f%%  |  Percentile: %.0fth\n",
		rec.CurrentRate,
		actionText,
		rec.ConfidenceScore,
		rec.PercentileRank,
	)

	if rec.OptimalWindow != nil && rec.Action == recommender.ActionWait {
		fmt.Printf("Better rate expected around %02d:00 CST (%.4f CNY predicted)\n",
			rec.OptimalWindow.StartHour,
			rec.OptimalWindow.ExpectedRate,
		)
	}

	fmt.Printf("\n")
	return nil
}

// DisplayHistoricalRanking shows where a rate ranks historically
func (c *RecommendCommand) DisplayHistoricalRanking(ctx context.Context, rate float64, days int) error {
	percentile, err := c.recommender.GetPercentileRank(ctx, rate, days)
	if err != nil {
		return fmt.Errorf("calculating percentile: %w", err)
	}

	ranking, err := c.recommender.GetHistoricalRanking(ctx, rate, days)
	if err != nil {
		return fmt.Errorf("getting ranking: %w", err)
	}

	fmt.Printf("\n")
	fmt.Printf("Historical Ranking\n")
	fmt.Printf("══════════════════\n")
	fmt.Printf("\n")
	fmt.Printf("Rate:        %.4f CNY\n", rate)
	fmt.Printf("Period:      Last %d days\n", days)
	fmt.Printf("Percentile:  %.0fth\n", percentile)
	fmt.Printf("Ranking:     %s\n", ranking)
	fmt.Printf("\n")

	// Visual percentile bar
	c.displayPercentileBar(percentile)
	fmt.Printf("\n")

	return nil
}

// displayPercentileBar shows a visual representation of percentile
func (c *RecommendCommand) displayPercentileBar(percentile float64) {
	barLength := 50
	filled := int((percentile / 100.0) * float64(barLength))

	fmt.Printf("  0%%  ")
	fmt.Printf("[")
	for i := 0; i < barLength; i++ {
		if i < filled {
			fmt.Printf("█")
		} else {
			fmt.Printf("░")
		}
	}
	fmt.Printf("]")
	fmt.Printf("  100%%\n")

	// Add pointer
	fmt.Printf("      ")
	for i := 0; i < filled; i++ {
		fmt.Printf(" ")
	}
	fmt.Printf("^\n")
}

// formatMoney formats a number as money with comma separators
func formatMoney(amount float64) string {
	// Simple formatting: add commas
	str := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(str, ".")

	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = "." + parts[1]
	}

	// Add commas to integer part
	var result strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	return result.String() + decPart
}

// formatTimeFromNow formats a future time as "in X hours/minutes"
func formatTimeFromNow(t time.Time) string {
	now := time.Now()
	if t.Before(now) {
		return "now"
	}

	duration := t.Sub(now)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("in %d days", days)
	} else if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("in %dh %dm", hours, minutes)
		}
		return fmt.Sprintf("in %d hours", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("in %d minutes", minutes)
	}

	return "in less than a minute"
}
