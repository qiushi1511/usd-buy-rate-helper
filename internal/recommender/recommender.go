package recommender

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/qiushi1511/usd-buy-rate-monitor/internal/storage"
)

// Action represents the recommended action
type Action string

const (
	ActionExchangeNow Action = "EXCHANGE_NOW" // High confidence to exchange immediately
	ActionWait        Action = "WAIT"         // Better rate likely in near future
	ActionNeutral     Action = "NEUTRAL"      // No strong signal either way
)

// Confidence represents the confidence level of the recommendation
type Confidence string

const (
	ConfidenceVeryHigh Confidence = "VERY_HIGH" // >85%
	ConfidenceHigh     Confidence = "HIGH"      // 70-85%
	ConfidenceMedium   Confidence = "MEDIUM"    // 50-70%
	ConfidenceLow      Confidence = "LOW"       // <50%
)

// Recommendation contains the exchange recommendation and supporting data
type Recommendation struct {
	Action          Action
	Confidence      Confidence
	ConfidenceScore float64 // 0-100
	CurrentRate     float64
	PercentileRank  float64 // 0-100, where 100 is best
	Amount          float64 // RMB amount
	USDAmount       float64 // Converted USD

	// Predictions
	PredictedNextHours []HourPrediction
	OptimalWindow      *TimeWindow

	// Analysis
	Reasoning       []string
	PotentialGain   float64 // USD gain if wait for optimal time
	PotentialLoss   float64 // USD loss if prediction wrong
	RiskLevel       string  // Low, Medium, High
	NextCheckTime   time.Time
	HistoricalStats HistoricalContext
}

// HourPrediction predicts rate for upcoming hours
type HourPrediction struct {
	Hour          int
	TimeLabel     string
	PredictedRate float64
	Confidence    float64
	Reasoning     string
}

// TimeWindow represents an optimal exchange time window
type TimeWindow struct {
	StartHour     int
	EndHour       int
	StartTime     time.Time
	EndTime       time.Time
	ExpectedRate  float64
	Probability   float64
	ReasoningText string
}

// HistoricalContext provides context about current rate vs history
type HistoricalContext struct {
	AvgRate30Days   float64
	MinRate30Days   float64
	MaxRate30Days   float64
	StdDev30Days    float64
	TodayDayOfWeek  string
	CurrentHour     int
	HourlyAvgRate   float64
	DailyAvgRate    float64
}

// Recommender provides intelligent exchange recommendations
type Recommender struct {
	repo   *storage.Repository
	logger *slog.Logger
}

// NewRecommender creates a new recommendation engine
func NewRecommender(repo *storage.Repository, logger *slog.Logger) *Recommender {
	return &Recommender{
		repo:   repo,
		logger: logger,
	}
}

// GetRecommendation analyzes current conditions and returns exchange recommendation
func (r *Recommender) GetRecommendation(ctx context.Context, amount float64) (*Recommendation, error) {
	// Get current rate
	latest, err := r.repo.GetLatestRate(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest rate: %w", err)
	}

	if latest == nil {
		return nil, fmt.Errorf("no rate data available")
	}

	currentRate := latest.RtcBid
	now := time.Now()

	// Get historical context (last 30 days)
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	historicalRates, err := r.repo.GetRatesByTimeRange(ctx, thirtyDaysAgo, now)
	if err != nil {
		return nil, fmt.Errorf("getting historical rates: %w", err)
	}

	if len(historicalRates) < 100 {
		return nil, fmt.Errorf("insufficient historical data (need at least 100 samples, have %d)", len(historicalRates))
	}

	// Calculate percentile rank
	percentile := r.calculatePercentile(currentRate, historicalRates)

	// Get hourly patterns
	hourlyPatterns, err := r.repo.GetHourlyPatterns(ctx, 30)
	if err != nil {
		return nil, fmt.Errorf("getting hourly patterns: %w", err)
	}

	// Get day of week patterns
	dowPatterns, err := r.repo.GetDayOfWeekPatterns(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("getting day of week patterns: %w", err)
	}

	// Build historical context
	histContext := r.buildHistoricalContext(historicalRates, hourlyPatterns, dowPatterns, now)

	// Generate predictions for next few hours
	predictions := r.predictNextHours(now, hourlyPatterns, currentRate, histContext)

	// Find optimal exchange window
	optimalWindow := r.findOptimalWindow(now, predictions, hourlyPatterns)

	// Determine action and confidence
	action, confidence, confidenceScore, reasoning := r.determineAction(
		currentRate,
		percentile,
		predictions,
		optimalWindow,
		histContext,
		now,
	)

	// Calculate potential gain/loss
	potentialGain, potentialLoss, riskLevel := r.calculateRiskReward(
		currentRate,
		amount,
		optimalWindow,
		histContext,
	)

	// Determine next check time
	nextCheck := r.determineNextCheckTime(now, action, optimalWindow)

	rec := &Recommendation{
		Action:             action,
		Confidence:         confidence,
		ConfidenceScore:    confidenceScore,
		CurrentRate:        currentRate,
		PercentileRank:     percentile,
		Amount:             amount,
		USDAmount:          amount / currentRate,
		PredictedNextHours: predictions,
		OptimalWindow:      optimalWindow,
		Reasoning:          reasoning,
		PotentialGain:      potentialGain,
		PotentialLoss:      potentialLoss,
		RiskLevel:          riskLevel,
		NextCheckTime:      nextCheck,
		HistoricalStats:    histContext,
	}

	return rec, nil
}

// calculatePercentile calculates where current rate ranks (0-100, higher is better)
func (r *Recommender) calculatePercentile(currentRate float64, historicalRates []storage.ExchangeRate) float64 {
	count := 0
	for _, rate := range historicalRates {
		if rate.RtcBid <= currentRate {
			count++
		}
	}
	return (float64(count) / float64(len(historicalRates))) * 100.0
}

// buildHistoricalContext creates context from historical data
func (r *Recommender) buildHistoricalContext(
	rates []storage.ExchangeRate,
	hourlyPatterns []storage.HourlyPattern,
	dowPatterns []storage.DayOfWeekPattern,
	now time.Time,
) HistoricalContext {
	// Calculate 30-day statistics
	var sum, sumSq float64
	min := math.MaxFloat64
	max := -math.MaxFloat64

	for _, rate := range rates {
		sum += rate.RtcBid
		sumSq += rate.RtcBid * rate.RtcBid
		if rate.RtcBid < min {
			min = rate.RtcBid
		}
		if rate.RtcBid > max {
			max = rate.RtcBid
		}
	}

	n := float64(len(rates))
	avg := sum / n
	variance := (sumSq / n) - (avg * avg)
	stdDev := math.Sqrt(variance)

	// Get current hour pattern
	cstLocation := time.FixedZone("CST", 8*60*60)
	nowCST := now.In(cstLocation)
	currentHour := nowCST.Hour()
	currentDOW := nowCST.Weekday()

	var hourlyAvg, dailyAvg float64
	for _, hp := range hourlyPatterns {
		if hp.Hour == currentHour {
			hourlyAvg = hp.AvgRate
			break
		}
	}

	for _, dp := range dowPatterns {
		if dp.DayOfWeek == int(currentDOW) {
			dailyAvg = dp.AvgRate
			break
		}
	}

	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	return HistoricalContext{
		AvgRate30Days:  avg,
		MinRate30Days:  min,
		MaxRate30Days:  max,
		StdDev30Days:   stdDev,
		TodayDayOfWeek: dayNames[currentDOW],
		CurrentHour:    currentHour,
		HourlyAvgRate:  hourlyAvg,
		DailyAvgRate:   dailyAvg,
	}
}

// predictNextHours generates predictions for next 4-6 hours
func (r *Recommender) predictNextHours(
	now time.Time,
	hourlyPatterns []storage.HourlyPattern,
	currentRate float64,
	histContext HistoricalContext,
) []HourPrediction {
	cstLocation := time.FixedZone("CST", 8*60*60)
	nowCST := now.In(cstLocation)
	currentHour := nowCST.Hour()

	predictions := []HourPrediction{}

	// Predict next 6 hours (or until end of business hours)
	for i := 1; i <= 6; i++ {
		futureHour := (currentHour + i) % 24
		if futureHour < 8 || futureHour >= 22 {
			break // Outside business hours
		}

		// Find pattern for this hour
		var pattern *storage.HourlyPattern
		for j := range hourlyPatterns {
			if hourlyPatterns[j].Hour == futureHour {
				pattern = &hourlyPatterns[j]
				break
			}
		}

		if pattern == nil {
			continue
		}

		// Predict rate: blend historical average with current trend
		// 70% historical pattern, 30% current rate adjustment
		trendAdjustment := currentRate - histContext.HourlyAvgRate
		predictedRate := (pattern.AvgRate * 0.7) + ((currentRate + trendAdjustment*0.3) * 0.3)

		// Confidence based on sample count and volatility
		confidence := 50.0
		if pattern.SampleCount > 100 {
			confidence += 20.0
		}
		rangeRatio := (pattern.MaxRate - pattern.MinRate) / pattern.AvgRate
		if rangeRatio < 0.01 {
			confidence += 15.0 // Low volatility increases confidence
		}
		if confidence > 85 {
			confidence = 85
		}

		reasoning := fmt.Sprintf("Based on %d samples", pattern.SampleCount)
		if pattern.PeakFreq > 0 {
			reasoning += fmt.Sprintf(", peak hour %.0f%% of time", float64(pattern.PeakFreq)/30.0*100.0)
		}

		predictions = append(predictions, HourPrediction{
			Hour:          futureHour,
			TimeLabel:     fmt.Sprintf("%02d:00", futureHour),
			PredictedRate: predictedRate,
			Confidence:    confidence,
			Reasoning:     reasoning,
		})
	}

	return predictions
}

// findOptimalWindow identifies the best time window to exchange
func (r *Recommender) findOptimalWindow(
	now time.Time,
	predictions []HourPrediction,
	hourlyPatterns []storage.HourlyPattern,
) *TimeWindow {
	if len(predictions) == 0 {
		return nil
	}

	// Find hour with highest predicted rate
	var bestPrediction *HourPrediction
	for i := range predictions {
		if bestPrediction == nil || predictions[i].PredictedRate > bestPrediction.PredictedRate {
			bestPrediction = &predictions[i]
		}
	}

	if bestPrediction == nil {
		return nil
	}

	// Create time window around optimal hour (±1 hour)
	cstLocation := time.FixedZone("CST", 8*60*60)
	nowCST := now.In(cstLocation)

	startHour := bestPrediction.Hour
	endHour := bestPrediction.Hour + 1
	if endHour >= 22 {
		endHour = 21
	}

	startTime := time.Date(nowCST.Year(), nowCST.Month(), nowCST.Day(), startHour, 0, 0, 0, cstLocation)
	endTime := time.Date(nowCST.Year(), nowCST.Month(), nowCST.Day(), endHour, 59, 59, 0, cstLocation)

	// If optimal time is tomorrow (after business hours today)
	if startTime.Before(nowCST) && nowCST.Hour() >= 20 {
		startTime = startTime.AddDate(0, 0, 1)
		endTime = endTime.AddDate(0, 0, 1)
	}

	reasoning := fmt.Sprintf("Historical data shows %02d:00-%02d:00 typically has higher rates", startHour, endHour)

	return &TimeWindow{
		StartHour:     startHour,
		EndHour:       endHour,
		StartTime:     startTime,
		EndTime:       endTime,
		ExpectedRate:  bestPrediction.PredictedRate,
		Probability:   bestPrediction.Confidence,
		ReasoningText: reasoning,
	}
}

// determineAction decides the recommended action
func (r *Recommender) determineAction(
	currentRate float64,
	percentile float64,
	predictions []HourPrediction,
	optimalWindow *TimeWindow,
	histContext HistoricalContext,
	now time.Time,
) (Action, Confidence, float64, []string) {
	reasons := []string{}
	score := 0.0

	// Factor 1: Percentile rank (40% weight)
	if percentile >= 90 {
		score += 40
		reasons = append(reasons, fmt.Sprintf("Current rate is at %.0fth percentile (excellent)", percentile))
	} else if percentile >= 75 {
		score += 30
		reasons = append(reasons, fmt.Sprintf("Current rate is at %.0fth percentile (good)", percentile))
	} else if percentile >= 50 {
		score += 15
		reasons = append(reasons, fmt.Sprintf("Current rate is at %.0fth percentile (average)", percentile))
	} else {
		score += 0
		reasons = append(reasons, fmt.Sprintf("Current rate is at %.0fth percentile (below average)", percentile))
	}

	// Factor 2: Comparison to hourly average (25% weight)
	if histContext.HourlyAvgRate > 0 {
		diff := currentRate - histContext.HourlyAvgRate
		diffPct := (diff / histContext.HourlyAvgRate) * 100

		if diff >= 0 {
			if diffPct > 0.5 {
				score += 25
				reasons = append(reasons, fmt.Sprintf("Rate is %.2f%% above hourly average", diffPct))
			} else {
				score += 15
				reasons = append(reasons, fmt.Sprintf("Rate matches hourly average"))
			}
		} else {
			score += 5
			reasons = append(reasons, fmt.Sprintf("Rate is %.2f%% below hourly average", -diffPct))
		}
	}

	// Factor 3: Future predictions (35% weight)
	if optimalWindow != nil {
		cstLocation := time.FixedZone("CST", 8*60*60)
		nowCST := now.In(cstLocation)

		// Is the optimal window in the near future?
		hoursUntilOptimal := optimalWindow.StartTime.Sub(nowCST).Hours()

		if hoursUntilOptimal <= 0 {
			// We're in the optimal window now
			score += 35
			reasons = append(reasons, "Currently in predicted optimal time window")
		} else if hoursUntilOptimal <= 3 {
			// Optimal window is soon
			expectedGain := optimalWindow.ExpectedRate - currentRate
			expectedGainPct := (expectedGain / currentRate) * 100

			if expectedGainPct > 0.3 {
				score += 5 // Wait for better rate
				reasons = append(reasons, fmt.Sprintf("Better rate predicted in %.0f hours (+%.2f%%)", hoursUntilOptimal, expectedGainPct))
			} else {
				score += 20 // Marginal improvement, exchange now OK
				reasons = append(reasons, fmt.Sprintf("Marginal improvement expected (%.0f hours, +%.2f%%)", hoursUntilOptimal, expectedGainPct))
			}
		} else {
			score += 10
			reasons = append(reasons, "No significant improvement predicted in near term")
		}
	}

	// Determine action based on score
	var action Action
	var confidence Confidence

	if score >= 75 {
		action = ActionExchangeNow
		if score >= 90 {
			confidence = ConfidenceVeryHigh
		} else if score >= 80 {
			confidence = ConfidenceHigh
		} else {
			confidence = ConfidenceMedium
		}
	} else if score >= 40 {
		action = ActionNeutral
		confidence = ConfidenceMedium
	} else {
		action = ActionWait
		if score < 25 {
			confidence = ConfidenceHigh
		} else {
			confidence = ConfidenceMedium
		}
	}

	return action, confidence, score, reasons
}

// calculateRiskReward estimates potential gain and loss
func (r *Recommender) calculateRiskReward(
	currentRate float64,
	amount float64,
	optimalWindow *TimeWindow,
	histContext HistoricalContext,
) (float64, float64, string) {
	currentUSD := amount / currentRate

	potentialGain := 0.0
	potentialLoss := 0.0
	riskLevel := "LOW"

	if optimalWindow != nil && optimalWindow.ExpectedRate > currentRate {
		// Gain if prediction is correct
		optentialUSD := amount / optimalWindow.ExpectedRate
		potentialGain = optentialUSD - currentUSD

		// Loss if rate goes down to recent min
		worstCaseRate := histContext.MinRate30Days
		worstCaseUSD := amount / worstCaseRate
		potentialLoss = currentUSD - worstCaseUSD

		// Risk assessment based on volatility
		volatility := histContext.StdDev30Days / histContext.AvgRate30Days
		if volatility > 0.02 {
			riskLevel = "HIGH"
		} else if volatility > 0.01 {
			riskLevel = "MEDIUM"
		}
	}

	return potentialGain, potentialLoss, riskLevel
}

// determineNextCheckTime suggests when to reassess
func (r *Recommender) determineNextCheckTime(now time.Time, action Action, optimalWindow *TimeWindow) time.Time {
	switch action {
	case ActionExchangeNow:
		// If recommending exchange now, suggest recheck in 30 minutes in case user delays
		return now.Add(30 * time.Minute)
	case ActionWait:
		// If waiting, check 30 min before optimal window
		if optimalWindow != nil {
			return optimalWindow.StartTime.Add(-30 * time.Minute)
		}
		return now.Add(1 * time.Hour)
	default:
		// Neutral: check in 1 hour
		return now.Add(1 * time.Hour)
	}
}

// GetPercentileRank returns the percentile rank of a given rate (public helper)
func (r *Recommender) GetPercentileRank(ctx context.Context, rate float64, days int) (float64, error) {
	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	rates, err := r.repo.GetRatesByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("getting historical rates: %w", err)
	}

	if len(rates) == 0 {
		return 0, fmt.Errorf("no historical data available")
	}

	return r.calculatePercentile(rate, rates), nil
}

// GetHistoricalRanking provides historical context for a rate
func (r *Recommender) GetHistoricalRanking(ctx context.Context, rate float64, days int) (string, error) {
	percentile, err := r.GetPercentileRank(ctx, rate, days)
	if err != nil {
		return "", err
	}

	if percentile >= 95 {
		return "Excellent (top 5%)", nil
	} else if percentile >= 85 {
		return "Very Good (top 15%)", nil
	} else if percentile >= 70 {
		return "Good (top 30%)", nil
	} else if percentile >= 50 {
		return "Average (median)", nil
	} else if percentile >= 30 {
		return "Below Average (bottom 70%)", nil
	} else {
		return "Poor (bottom 30%)", nil
	}
}

// SortRatesByValue sorts rates by value in descending order (for percentile calculations)
func sortRatesByValue(rates []storage.ExchangeRate) []storage.ExchangeRate {
	sorted := make([]storage.ExchangeRate, len(rates))
	copy(sorted, rates)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RtcBid > sorted[j].RtcBid
	})
	return sorted
}
