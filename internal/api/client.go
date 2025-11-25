package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"time"
)

const (
	defaultBaseURL    = "https://m.cmbchina.com/api/rate/fx-rate"
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 2 * time.Second
)

// Client is an HTTP client for fetching exchange rates from CMB API
type Client struct {
	httpClient *http.Client
	baseURL    string
	maxRetries int
	retryDelay time.Duration
	logger     *slog.Logger
}

// NewClient creates a new API client with default configuration
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:    defaultBaseURL,
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
		logger:     logger,
	}
}

// FetchExchangeRates retrieves current exchange rates with retry logic
func (c *Client) FetchExchangeRates(ctx context.Context) (*CMBResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := c.calculateBackoff(attempt)
			c.logger.Info("retrying API request",
				"attempt", attempt,
				"backoff", backoff)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.fetchOnce(ctx)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on certain errors
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode < 500 {
			// Don't retry 4xx client errors
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}

		c.logger.Warn("API request failed",
			"attempt", attempt+1,
			"error", err)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// fetchOnce performs a single API request
func (c *Client) fetchOnce(ctx context.Context) (*CMBResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "USD-Buy-Rate-Monitor/1.0")
	req.Header.Set("Accept", "application/json")

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		return nil, &NetworkError{Err: err, Duration: elapsed}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var cmbResp CMBResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmbResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	c.logger.Debug("API request successful",
		"response_time_ms", elapsed.Milliseconds(),
		"return_code", cmbResp.ReturnCode)

	return &cmbResp, nil
}

// calculateBackoff calculates exponential backoff with jitter
func (c *Client) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * 2^(attempt-1)
	base := c.retryDelay
	backoff := base * time.Duration(1<<uint(attempt-1))

	// Cap at 30 seconds
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	// Add jitter (±20%)
	jitter := time.Duration(rand.Int63n(int64(backoff) / 5))
	return backoff + jitter - (jitter / 2)
}

// NetworkError represents a network-level error
type NetworkError struct {
	Err      error
	Duration time.Duration
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error after %v: %v", e.Duration, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}
