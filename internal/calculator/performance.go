package calculator

import (
	"math"
	"sort"
)

// CalculatePercentChange calculates the percentage change from old to new price.
func CalculatePercentChange(old, new float64) float64 {
	if old == 0 {
		return 0
	}
	return ((new - old) / old) * 100
}

// CalculateMovingAverage calculates the simple moving average for the last N prices.
// Returns 0 if there are fewer prices than the period.
func CalculateMovingAverage(prices []float64, period int) float64 {
	if len(prices) < period || period <= 0 {
		return 0
	}

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// CalculateRSI calculates the Relative Strength Index over the specified period.
// Uses the standard Wilder smoothing method (exponential moving average).
func CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 || period <= 0 {
		return 50 // Default neutral RSI
	}

	// Calculate price changes
	changes := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes[i-1] = prices[i] - prices[i-1]
	}

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		if changes[i] > 0 {
			avgGain += changes[i]
		} else {
			avgLoss += -changes[i]
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Apply Wilder smoothing for remaining periods
	for i := period; i < len(changes); i++ {
		if changes[i] > 0 {
			avgGain = (avgGain*float64(period-1) + changes[i]) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) - changes[i]) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// CalculateVolatility calculates the annualized volatility (standard deviation of returns).
// Assumes daily prices and 252 trading days per year.
func CalculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// Calculate daily returns
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] > 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// Calculate mean return
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate variance
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	// Standard deviation * sqrt(252) for annualization
	return math.Sqrt(variance) * math.Sqrt(252) * 100
}

// Calculate52WeekRange finds the highest and lowest prices in the dataset.
func Calculate52WeekRange(prices []float64) (high, low float64) {
	if len(prices) == 0 {
		return 0, 0
	}

	sorted := make([]float64, len(prices))
	copy(sorted, prices)
	sort.Float64s(sorted)

	// Filter out zero/invalid prices
	validPrices := make([]float64, 0, len(sorted))
	for _, p := range sorted {
		if p > 0 {
			validPrices = append(validPrices, p)
		}
	}

	if len(validPrices) == 0 {
		return 0, 0
	}

	return validPrices[len(validPrices)-1], validPrices[0]
}

// FindPriceAtDaysAgo finds the price at approximately N days ago from the price slice.
// Returns 0 if not enough data.
func FindPriceAtDaysAgo(prices []float64, daysAgo int) float64 {
	if len(prices) == 0 {
		return 0
	}
	if daysAgo <= 0 {
		return prices[len(prices)-1]
	}
	idx := len(prices) - 1 - daysAgo
	if idx < 0 {
		idx = 0
	}
	return prices[idx]
}

// FindYTDStartPrice finds the price at the start of the year.
// Assumes prices are ordered chronologically and cover at least YTD period.
// daysIntoYear is the number of days since Jan 1.
func FindYTDStartPrice(prices []float64, daysIntoYear int) float64 {
	if len(prices) == 0 {
		return 0
	}
	if daysIntoYear <= 0 || daysIntoYear >= len(prices) {
		return prices[0]
	}
	idx := len(prices) - daysIntoYear
	if idx < 0 {
		idx = 0
	}
	return prices[idx]
}
