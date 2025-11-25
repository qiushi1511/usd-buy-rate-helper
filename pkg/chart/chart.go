package chart

import (
	"fmt"

	"github.com/guptarohit/asciigraph"
	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// RenderLineChart creates an ASCII line chart from exchange rate data
func RenderLineChart(rates []storage.ExchangeRate, width, height int) string {
	if len(rates) == 0 {
		return "No data to display"
	}

	// Extract rate values for the chart
	data := make([]float64, len(rates))
	for i, rate := range rates {
		data[i] = rate.RtcBid
	}

	// Configure chart
	graph := asciigraph.Plot(data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Caption(fmt.Sprintf("USD/CNY Rate (%s to %s)",
			rates[0].CollectedAt.Format("15:04"),
			rates[len(rates)-1].CollectedAt.Format("15:04"))),
	)

	return graph
}

// RenderLineChartWithLabels creates a chart with custom X-axis labels
func RenderLineChartWithLabels(rates []storage.ExchangeRate, width, height int, showEvery int) string {
	if len(rates) == 0 {
		return "No data to display"
	}

	// Extract rate values
	data := make([]float64, len(rates))
	for i, rate := range rates {
		data[i] = rate.RtcBid
	}

	// Calculate min and max for better visualization
	var min, max float64
	min = data[0]
	max = data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Configure chart with custom options
	graph := asciigraph.Plot(data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Caption(fmt.Sprintf("USD/CNY Exchange Rate Trend (%d samples)",
			len(rates))),
		asciigraph.Precision(4),
	)

	return graph
}

// RenderDailyAverageChart creates a chart for daily average rates
func RenderDailyAverageChart(stats []*storage.DailyStats, width, height int) string {
	if len(stats) == 0 {
		return "No data to display"
	}

	// Extract average rates
	data := make([]float64, len(stats))
	for i, stat := range stats {
		data[i] = stat.AvgRate
	}

	// Create caption with date range
	caption := fmt.Sprintf("Daily Average Rates (%s to %s)",
		stats[len(stats)-1].Date,
		stats[0].Date)

	graph := asciigraph.Plot(data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Caption(caption),
		asciigraph.Precision(4),
	)

	return graph
}

// RenderVolatilityChart shows the daily volatility (max-min range)
func RenderVolatilityChart(stats []*storage.DailyStats, width, height int) string {
	if len(stats) == 0 {
		return "No data to display"
	}

	// Extract volatility (range) values
	data := make([]float64, len(stats))
	for i, stat := range stats {
		data[i] = stat.MaxRate - stat.MinRate
	}

	caption := fmt.Sprintf("Daily Volatility (%s to %s)",
		stats[len(stats)-1].Date,
		stats[0].Date)

	graph := asciigraph.Plot(data,
		asciigraph.Width(width),
		asciigraph.Height(height),
		asciigraph.Caption(caption),
		asciigraph.Precision(4),
	)

	return graph
}

// GetTerminalDimensions returns recommended chart dimensions
// This is a simple heuristic - could be made smarter with terminal size detection
func GetTerminalDimensions() (width, height int) {
	// Default dimensions that work well in most terminals
	return 70, 15
}

// RenderMiniChart creates a small inline chart for quick viewing
func RenderMiniChart(rates []storage.ExchangeRate) string {
	if len(rates) == 0 {
		return ""
	}

	data := make([]float64, len(rates))
	for i, rate := range rates {
		data[i] = rate.RtcBid
	}

	// Mini chart with reduced dimensions
	graph := asciigraph.Plot(data,
		asciigraph.Width(40),
		asciigraph.Height(8),
		asciigraph.Precision(4),
	)

	return graph
}

// FormatTimeLabels creates time labels for X-axis
func FormatTimeLabels(rates []storage.ExchangeRate, count int) []string {
	if len(rates) == 0 || count <= 0 {
		return nil
	}

	labels := make([]string, 0, count)
	step := len(rates) / count
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(rates); i += step {
		if len(labels) >= count {
			break
		}
		labels = append(labels, rates[i].CollectedAt.Format("15:04"))
	}

	return labels
}

// PrintChartWithStats prints a chart along with statistical summary
func PrintChartWithStats(rates []storage.ExchangeRate, width, height int) {
	if len(rates) == 0 {
		fmt.Println("No data available")
		return
	}

	// Calculate statistics
	var sum, min, max float64
	min = rates[0].RtcBid
	max = rates[0].RtcBid

	for _, rate := range rates {
		sum += rate.RtcBid
		if rate.RtcBid < min {
			min = rate.RtcBid
		}
		if rate.RtcBid > max {
			max = rate.RtcBid
		}
	}
	avg := sum / float64(len(rates))

	// Print chart
	fmt.Println()
	fmt.Println(RenderLineChart(rates, width, height))
	fmt.Println()

	// Print statistics below chart
	fmt.Printf("Statistics: Min=%.4f  Max=%.4f  Avg=%.4f  Range=%.4f  Samples=%d\n",
		min, max, avg, max-min, len(rates))
	fmt.Println()
}
