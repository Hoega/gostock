package model

// AssetPerformance holds performance metrics for a single asset (stock or crypto).
type AssetPerformance struct {
	ID           string  // Internal ID (ISIN for stocks, CoingeckoID for crypto)
	Name         string  // Display name
	Symbol       string  // Ticker symbol
	Type         string  // "stock" or "crypto"
	CurrentPrice float64 // Current price in EUR
	TotalValue   float64 // Total position value in EUR
	Quantity     float64 // Number of units held

	// Price performance (% changes)
	DailyChange   float64
	WeeklyChange  float64
	MonthlyChange float64
	YTDChange     float64
	YearChange    float64

	// 52-week range
	High52Week float64
	Low52Week  float64

	// Technical indicators
	MA50       float64 // 50-day moving average
	MA200      float64 // 200-day moving average
	RSI14      float64 // 14-day RSI
	Volatility float64 // Annualized volatility
}

// RangePercent returns the position of current price within 52-week range (0-100).
func (a AssetPerformance) RangePercent() float64 {
	if a.High52Week == a.Low52Week {
		return 50
	}
	pct := (a.CurrentPrice - a.Low52Week) / (a.High52Week - a.Low52Week) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// PerformanceDashboardData holds all data for the dashboard page.
type PerformanceDashboardData struct {
	Assets         []AssetPerformance
	Filter         string  // "all", "stocks", "crypto"
	SortBy         string  // Column to sort by
	SortDesc       bool    // Sort descending
	BestPerformer  string  // Name of best performer (by YTD)
	WorstPerformer string  // Name of worst performer (by YTD)
	AverageReturn  float64 // Average YTD return across all assets
	TotalValue     float64 // Total portfolio value
	StockCount     int     // Number of stock positions
	CryptoCount    int     // Number of crypto positions
	UpdatedAt      string  // Last data fetch timestamp (formatted)
}
