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
	apiClient *api.Client
	repo      *storage.Repository
	logger    *slog.Logger
}

// NewPoller creates a new poller instance
func NewPoller(apiClient *api.Client, repo *storage.Repository, logger *slog.Logger) *Poller {
	return &Poller{
		apiClient: apiClient,
		repo:      repo,
		logger:    logger,
	}
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

// poll performs a single polling operation
func (p *Poller) poll(ctx context.Context) error {
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
