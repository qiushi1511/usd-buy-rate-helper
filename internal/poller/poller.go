package poller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/api"
	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// Poller handles periodic polling of exchange rates
type Poller struct {
	apiClient           *api.Client
	repo                *storage.Repository
	logger              *slog.Logger
	skipOffHours        bool
	businessHoursStart  int // Hour in CST (0-23)
	businessHoursEnd    int // Hour in CST (0-23)
}

// PollerOption configures the poller
type PollerOption func(*Poller)

// WithBusinessHours enables business hours checking
func WithBusinessHours(start, end int) PollerOption {
	return func(p *Poller) {
		p.skipOffHours = true
		p.businessHoursStart = start
		p.businessHoursEnd = end
	}
}

// WithoutBusinessHours disables business hours checking (poll 24/7)
func WithoutBusinessHours() PollerOption {
	return func(p *Poller) {
		p.skipOffHours = false
	}
}

// NewPoller creates a new poller instance
func NewPoller(apiClient *api.Client, repo *storage.Repository, logger *slog.Logger, opts ...PollerOption) *Poller {
	p := &Poller{
		apiClient:          apiClient,
		repo:               repo,
		logger:             logger,
		skipOffHours:       true,  // Default: enable business hours check
		businessHoursStart: 8,      // Default: 08:00 CST
		businessHoursEnd:   22,     // Default: 22:00 CST
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Start begins the polling loop with the specified interval
func (p *Poller) Start(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	p.logger.Info("poller started", "interval", interval)

	// Perform initial poll immediately
	if err := p.poll(ctx); err != nil {
		p.logger.Error("initial poll failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poller stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				p.logger.Error("poll failed", "error", err)
				// Continue polling despite errors
			}
		}
	}
}

// isBusinessHours checks if the current time is within CMB business hours (CST)
func (p *Poller) isBusinessHours() bool {
	if !p.skipOffHours {
		return true // Always poll if business hours check is disabled
	}

	// Get current time in CST (China Standard Time = UTC+8)
	cstLocation := time.FixedZone("CST", 8*60*60)
	now := time.Now().In(cstLocation)
	hour := now.Hour()

	return hour >= p.businessHoursStart && hour < p.businessHoursEnd
}

// poll performs a single polling operation
func (p *Poller) poll(ctx context.Context) error {
	// Check if we're within business hours
	if !p.isBusinessHours() {
		cstLocation := time.FixedZone("CST", 8*60*60)
		now := time.Now().In(cstLocation)
		p.logger.Debug("skipping poll outside business hours",
			"current_hour_cst", now.Hour(),
			"business_hours", fmt.Sprintf("%02d:00-%02d:00", p.businessHoursStart, p.businessHoursEnd))
		return nil
	}

	startTime := time.Now()

	// Fetch data from API
	resp, err := p.apiClient.FetchExchangeRates(ctx)
	if err != nil {
		return fmt.Errorf("fetching rates: %w", err)
	}

	// Extract USD rate
	usdRate, err := api.ExtractUSDRate(resp)
	if err != nil {
		return fmt.Errorf("extracting USD rate: %w", err)
	}

	// Store rate in database
	rate := &storage.ExchangeRate{
		CurrencyCode:  "USD",
		RtcBid:        usdRate,
		CollectedAt:   startTime,
		DatePartition: startTime.Format("2006-01-02"),
	}

	if err := p.repo.InsertRate(ctx, rate); err != nil {
		return fmt.Errorf("storing rate: %w", err)
	}

	elapsed := time.Since(startTime)
	p.logger.Info("poll successful",
		"rate", usdRate,
		"elapsed_ms", elapsed.Milliseconds())

	return nil
}
