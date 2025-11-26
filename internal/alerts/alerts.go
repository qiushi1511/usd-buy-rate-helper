package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeThresholdHigh  AlertType = "threshold_high"
	AlertTypeThresholdLow   AlertType = "threshold_low"
	AlertTypeChangeIncrease AlertType = "change_increase"
	AlertTypeChangeDecrease AlertType = "change_decrease"
	AlertTypeUnusual        AlertType = "unusual_pattern"
)

// Alert represents an alert condition
type Alert struct {
	Type      AlertType
	Message   string
	Rate      float64
	Threshold float64
	Change    float64
	Timestamp time.Time
}

// Config holds alert configuration
type Config struct {
	HighThreshold      float64 // Alert if rate goes above this
	LowThreshold       float64 // Alert if rate goes below this
	ChangePercent      float64 // Alert if rate changes by this % in short time
	CheckPatterns      bool    // Alert on unusual patterns (deviation from historical)
	PatternStdDevs     float64 // Number of std deviations for pattern alerts
	CooldownMinutes    int     // Minutes to wait before repeating same alert
}

// Manager handles alert checking and notifications
type Manager struct {
	config       *Config
	repo         *storage.Repository
	logger       *slog.Logger
	lastAlerts   map[AlertType]time.Time // Track last alert time per type
	lastRate     float64
	lastRateTime time.Time
}

// NewManager creates a new alert manager
func NewManager(config *Config, repo *storage.Repository, logger *slog.Logger) *Manager {
	return &Manager{
		config:     config,
		repo:       repo,
		logger:     logger,
		lastAlerts: make(map[AlertType]time.Time),
	}
}

// Check examines a new rate for alert conditions
func (m *Manager) Check(ctx context.Context, rate float64, timestamp time.Time) []Alert {
	var alerts []Alert

	// Check threshold alerts
	if m.config.HighThreshold > 0 && rate > m.config.HighThreshold {
		if m.shouldAlert(AlertTypeThresholdHigh) {
			alerts = append(alerts, Alert{
				Type:      AlertTypeThresholdHigh,
				Message:   fmt.Sprintf("Rate exceeded high threshold: %.4f > %.4f CNY", rate, m.config.HighThreshold),
				Rate:      rate,
				Threshold: m.config.HighThreshold,
				Timestamp: timestamp,
			})
			m.markAlerted(AlertTypeThresholdHigh)
		}
	}

	if m.config.LowThreshold > 0 && rate < m.config.LowThreshold {
		if m.shouldAlert(AlertTypeThresholdLow) {
			alerts = append(alerts, Alert{
				Type:      AlertTypeThresholdLow,
				Message:   fmt.Sprintf("Rate dropped below low threshold: %.4f < %.4f CNY", rate, m.config.LowThreshold),
				Rate:      rate,
				Threshold: m.config.LowThreshold,
				Timestamp: timestamp,
			})
			m.markAlerted(AlertTypeThresholdLow)
		}
	}

	// Check change alerts (compared to last rate)
	if m.lastRate > 0 && m.config.ChangePercent > 0 {
		changePercent := ((rate - m.lastRate) / m.lastRate) * 100
		absChange := changePercent
		if absChange < 0 {
			absChange = -absChange
		}

		if absChange >= m.config.ChangePercent {
			alertType := AlertTypeChangeIncrease
			direction := "increased"
			if changePercent < 0 {
				alertType = AlertTypeChangeDecrease
				direction = "decreased"
			}

			if m.shouldAlert(alertType) {
				timeDiff := timestamp.Sub(m.lastRateTime)
				alerts = append(alerts, Alert{
					Type:      alertType,
					Message:   fmt.Sprintf("Rate %s by %.2f%% in %v: %.4f → %.4f CNY", direction, absChange, timeDiff.Round(time.Minute), m.lastRate, rate),
					Rate:      rate,
					Change:    changePercent,
					Timestamp: timestamp,
				})
				m.markAlerted(alertType)
			}
		}
	}

	// Check pattern-based alerts
	if m.config.CheckPatterns {
		patternAlert := m.checkPatternDeviation(ctx, rate, timestamp)
		if patternAlert != nil {
			alerts = append(alerts, *patternAlert)
		}
	}

	// Update last rate
	m.lastRate = rate
	m.lastRateTime = timestamp

	return alerts
}

// checkPatternDeviation checks if current rate is unusual compared to historical patterns
func (m *Manager) checkPatternDeviation(ctx context.Context, rate float64, timestamp time.Time) *Alert {
	if !m.shouldAlert(AlertTypeUnusual) {
		return nil
	}

	// Get hourly pattern for this hour
	hour := timestamp.Hour()
	patterns, err := m.repo.GetHourlyPatterns(ctx, 30) // Last 30 days
	if err != nil {
		m.logger.Warn("failed to get hourly patterns for alert", "error", err)
		return nil
	}

	// Find pattern for current hour
	var hourPattern *storage.HourlyPattern
	for i := range patterns {
		if patterns[i].Hour == hour {
			hourPattern = &patterns[i]
			break
		}
	}

	if hourPattern == nil || hourPattern.SampleCount < 10 {
		return nil // Not enough data
	}

	// Calculate standard deviation (approximate from range)
	// For normal distribution, range ≈ 6 * stddev
	rng := hourPattern.MaxRate - hourPattern.MinRate
	stdDev := rng / 6.0

	if stdDev == 0 {
		return nil // No variation
	}

	// Check how many standard deviations away from average
	deviation := (rate - hourPattern.AvgRate) / stdDev
	absDeviation := deviation
	if absDeviation < 0 {
		absDeviation = -absDeviation
	}

	if absDeviation >= m.config.PatternStdDevs {
		direction := "higher"
		if deviation < 0 {
			direction = "lower"
		}

		m.markAlerted(AlertTypeUnusual)
		return &Alert{
			Type:      AlertTypeUnusual,
			Message:   fmt.Sprintf("Unusual rate at %02d:00: %.4f CNY is %.1f std devs %s than usual (avg: %.4f)", hour, rate, absDeviation, direction, hourPattern.AvgRate),
			Rate:      rate,
			Threshold: hourPattern.AvgRate,
			Timestamp: timestamp,
		}
	}

	return nil
}

// shouldAlert checks if we should send an alert based on cooldown
func (m *Manager) shouldAlert(alertType AlertType) bool {
	if m.config.CooldownMinutes <= 0 {
		return true
	}

	lastAlert, exists := m.lastAlerts[alertType]
	if !exists {
		return true
	}

	cooldown := time.Duration(m.config.CooldownMinutes) * time.Minute
	return time.Since(lastAlert) >= cooldown
}

// markAlerted records that an alert was sent
func (m *Manager) markAlerted(alertType AlertType) {
	m.lastAlerts[alertType] = time.Now()
}

// Notifier handles alert notifications
type Notifier interface {
	Notify(alert Alert) error
}

// LogNotifier logs alerts using slog
type LogNotifier struct {
	logger *slog.Logger
}

// NewLogNotifier creates a notifier that logs alerts
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

// Notify logs the alert
func (n *LogNotifier) Notify(alert Alert) error {
	n.logger.Warn("ALERT",
		"type", alert.Type,
		"message", alert.Message,
		"rate", alert.Rate,
		"timestamp", alert.Timestamp.Format("2006-01-02 15:04:05"),
	)
	return nil
}
