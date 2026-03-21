package handler

import (
	"html/template"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
	"github.com/Hoega/gostock/internal/quote"
)

// DashboardHandler handles the performance dashboard page.
type DashboardHandler struct {
	templates *template.Template
	store     persistence.Store
	cache     *dashboardCache
}

type dashboardCache struct {
	mu        sync.RWMutex
	data      []model.AssetPerformance
	fetchedAt time.Time
}

const dashboardCacheDuration = 15 * time.Minute

func NewDashboardHandler(templates *template.Template, store persistence.Store) *DashboardHandler {
	return &DashboardHandler{
		templates: templates,
		store:     store,
		cache:     &dashboardCache{},
	}
}

// ShowDashboard handles GET /dashboard.
func (h *DashboardHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = "all"
	}
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "value"
	}
	sortDesc := r.URL.Query().Get("desc") != "false"

	assets := h.getAssets()

	// Filter assets
	filtered := filterAssets(assets, filter)

	// Sort assets
	sortAssets(filtered, sortBy, sortDesc)

	// Compute summary stats
	data := buildDashboardData(filtered, filter, sortBy, sortDesc)

	// Set last update time
	h.cache.mu.RLock()
	if !h.cache.fetchedAt.IsZero() {
		data.UpdatedAt = h.cache.fetchedAt.Format("02/01/2006 à 15:04")
	}
	h.cache.mu.RUnlock()

	// Count totals from unfiltered data
	for _, a := range assets {
		if a.Type == "stock" {
			data.StockCount++
		} else {
			data.CryptoCount++
		}
		data.TotalValue += a.TotalValue
	}

	// Render response
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "dashboard-content.html", data); err != nil {
			log.Printf("Dashboard content template error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Dashboard template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getAssets returns cached assets or fetches fresh data.
func (h *DashboardHandler) getAssets() []model.AssetPerformance {
	h.cache.mu.RLock()
	if h.cache.data != nil && time.Since(h.cache.fetchedAt) < dashboardCacheDuration {
		result := make([]model.AssetPerformance, len(h.cache.data))
		copy(result, h.cache.data)
		h.cache.mu.RUnlock()
		return result
	}
	h.cache.mu.RUnlock()

	// Fetch fresh data
	assets := h.fetchAllAssets()

	h.cache.mu.Lock()
	h.cache.data = assets
	h.cache.fetchedAt = time.Now()
	h.cache.mu.Unlock()

	result := make([]model.AssetPerformance, len(assets))
	copy(result, assets)
	return result
}

// fetchAllAssets fetches performance data for all positions.
func (h *DashboardHandler) fetchAllAssets() []model.AssetPerformance {
	var assets []model.AssetPerformance
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Load stock positions
	stockPositions, err := h.store.LoadPositions()
	if err != nil {
		log.Printf("Failed to load stock positions: %v", err)
	}

	// Load crypto positions
	cryptoPositions, err := h.store.LoadCryptoPositions()
	if err != nil {
		log.Printf("Failed to load crypto positions: %v", err)
	}

	// Fetch EUR/USD exchange rate for USD stocks
	var usdRate float64 = 1.0
	needUSD := false
	for _, p := range stockPositions {
		if p.Currency == "USD" {
			needUSD = true
			break
		}
	}
	if needUSD {
		if rate, err := quote.FetchExchangeRate("EUR", "USD"); err == nil && rate > 0 {
			usdRate = 1.0 / rate
		}
	}

	// Fetch stock performance data in parallel
	for _, pos := range stockPositions {
		wg.Add(1)
		go func(p persistence.StockPosition, rate float64) {
			defer wg.Done()
			asset := h.fetchStockPerformance(p, rate)
			if asset != nil {
				mu.Lock()
				assets = append(assets, *asset)
				mu.Unlock()
			}
		}(pos, usdRateForCurrency(pos.Currency, usdRate))
	}

	// Fetch crypto performance data in parallel
	for _, pos := range cryptoPositions {
		wg.Add(1)
		go func(p persistence.CryptoPosition) {
			defer wg.Done()
			asset := h.fetchCryptoPerformance(p)
			if asset != nil {
				mu.Lock()
				assets = append(assets, *asset)
				mu.Unlock()
			}
		}(pos)
	}

	wg.Wait()
	return assets
}

func usdRateForCurrency(currency string, usdRate float64) float64 {
	if currency == "USD" {
		return usdRate
	}
	return 1.0
}

// fetchStockPerformance fetches and calculates performance for a stock position.
// Returns a basic asset from DB data if Yahoo calls fail, so the asset still appears.
func (h *DashboardHandler) fetchStockPerformance(pos persistence.StockPosition, eurRate float64) *model.AssetPerformance {
	symbol, err := quote.ResolveISINToSymbol(pos.ISIN)
	if err != nil {
		log.Printf("Resolve ISIN %s failed: %v", pos.ISIN, err)
		// Return basic asset from DB data so it still appears on dashboard.
		currentPriceEUR := pos.CurrentPrice * eurRate
		return &model.AssetPerformance{
			ID:           pos.ISIN,
			Name:         pos.Name,
			Symbol:       pos.ISIN,
			Type:         "stock",
			CurrentPrice: currentPriceEUR,
			TotalValue:   currentPriceEUR * pos.Quantity,
			Quantity:     pos.Quantity,
		}
	}

	history, err := quote.FetchStockHistory(symbol, "1y")
	if err != nil {
		log.Printf("Stock history %s failed: %v", symbol, err)
		// Return basic asset from DB data so it still appears on dashboard.
		currentPriceEUR := pos.CurrentPrice * eurRate
		return &model.AssetPerformance{
			ID:           pos.ISIN,
			Name:         pos.Name,
			Symbol:       symbol,
			Type:         "stock",
			CurrentPrice: currentPriceEUR,
			TotalValue:   currentPriceEUR * pos.Quantity,
			Quantity:     pos.Quantity,
		}
	}

	if len(history) == 0 {
		currentPriceEUR := pos.CurrentPrice * eurRate
		return &model.AssetPerformance{
			ID:           pos.ISIN,
			Name:         pos.Name,
			Symbol:       symbol,
			Type:         "stock",
			CurrentPrice: currentPriceEUR,
			TotalValue:   currentPriceEUR * pos.Quantity,
			Quantity:     pos.Quantity,
		}
	}

	prices := extractPrices(history)
	currentPrice := prices[len(prices)-1]
	currentPriceEUR := currentPrice * eurRate

	asset := &model.AssetPerformance{
		ID:           pos.ISIN,
		Name:         pos.Name,
		Symbol:       symbol,
		Type:         "stock",
		CurrentPrice: currentPriceEUR,
		TotalValue:   currentPriceEUR * pos.Quantity,
		Quantity:     pos.Quantity,
	}

	// Calculate performance metrics
	calculatePerformanceMetrics(asset, prices, currentPrice)

	// Convert 52-week range to EUR
	asset.High52Week *= eurRate
	asset.Low52Week *= eurRate
	asset.MA50 *= eurRate
	asset.MA200 *= eurRate

	return asset
}

// fetchCryptoPerformance fetches and calculates performance for a crypto position.
func (h *DashboardHandler) fetchCryptoPerformance(pos persistence.CryptoPosition) *model.AssetPerformance {
	history, err := quote.FetchCryptoHistory(pos.CoingeckoID, "1y")
	if err != nil {
		log.Printf("Crypto history %s failed: %v", pos.CoingeckoID, err)
		return nil
	}

	if len(history) == 0 {
		return nil
	}

	prices := extractPrices(history)
	currentPrice := prices[len(prices)-1]

	asset := &model.AssetPerformance{
		ID:           pos.CoingeckoID,
		Name:         pos.Name,
		Symbol:       pos.Symbol,
		Type:         "crypto",
		CurrentPrice: currentPrice,
		TotalValue:   currentPrice * pos.Quantity,
		Quantity:     pos.Quantity,
	}

	calculatePerformanceMetrics(asset, prices, currentPrice)

	return asset
}

// extractPrices extracts price values from PricePoint slice.
func extractPrices(history []quote.PricePoint) []float64 {
	prices := make([]float64, len(history))
	for i, pt := range history {
		prices[i] = pt.Price
	}
	return prices
}

// calculatePerformanceMetrics calculates all performance metrics for an asset.
func calculatePerformanceMetrics(asset *model.AssetPerformance, prices []float64, currentPrice float64) {
	n := len(prices)
	if n == 0 {
		return
	}

	// Calculate % changes
	if n >= 2 {
		asset.DailyChange = calculator.CalculatePercentChange(prices[n-2], currentPrice)
	}
	if n >= 7 {
		asset.WeeklyChange = calculator.CalculatePercentChange(calculator.FindPriceAtDaysAgo(prices, 7), currentPrice)
	}
	if n >= 30 {
		asset.MonthlyChange = calculator.CalculatePercentChange(calculator.FindPriceAtDaysAgo(prices, 30), currentPrice)
	}

	// YTD change - calculate days since Jan 1
	now := time.Now()
	daysIntoYear := now.YearDay()
	if n >= daysIntoYear {
		ytdPrice := calculator.FindYTDStartPrice(prices, daysIntoYear)
		if ytdPrice > 0 {
			asset.YTDChange = calculator.CalculatePercentChange(ytdPrice, currentPrice)
		}
	} else if n > 0 {
		// Use first available price if we don't have full YTD
		asset.YTDChange = calculator.CalculatePercentChange(prices[0], currentPrice)
	}

	// 1-year change
	if n >= 252 {
		asset.YearChange = calculator.CalculatePercentChange(prices[0], currentPrice)
	} else if n > 0 {
		asset.YearChange = calculator.CalculatePercentChange(prices[0], currentPrice)
	}

	// 52-week range
	asset.High52Week, asset.Low52Week = calculator.Calculate52WeekRange(prices)

	// Technical indicators
	asset.MA50 = calculator.CalculateMovingAverage(prices, 50)
	asset.MA200 = calculator.CalculateMovingAverage(prices, 200)
	asset.RSI14 = calculator.CalculateRSI(prices, 14)
	asset.Volatility = calculator.CalculateVolatility(prices)
}

// filterAssets filters assets by type.
func filterAssets(assets []model.AssetPerformance, filter string) []model.AssetPerformance {
	if filter == "all" {
		return assets
	}

	var filtered []model.AssetPerformance
	for _, a := range assets {
		if (filter == "stocks" && a.Type == "stock") || (filter == "crypto" && a.Type == "crypto") {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// sortAssets sorts assets by the specified column.
func sortAssets(assets []model.AssetPerformance, sortBy string, desc bool) {
	sort.Slice(assets, func(i, j int) bool {
		var vi, vj float64
		switch sortBy {
		case "name":
			if desc {
				return assets[i].Name > assets[j].Name
			}
			return assets[i].Name < assets[j].Name
		case "value":
			vi, vj = assets[i].TotalValue, assets[j].TotalValue
		case "price":
			vi, vj = assets[i].CurrentPrice, assets[j].CurrentPrice
		case "daily":
			vi, vj = assets[i].DailyChange, assets[j].DailyChange
		case "weekly":
			vi, vj = assets[i].WeeklyChange, assets[j].WeeklyChange
		case "monthly":
			vi, vj = assets[i].MonthlyChange, assets[j].MonthlyChange
		case "ytd":
			vi, vj = assets[i].YTDChange, assets[j].YTDChange
		case "year":
			vi, vj = assets[i].YearChange, assets[j].YearChange
		case "rsi":
			vi, vj = assets[i].RSI14, assets[j].RSI14
		case "volatility":
			vi, vj = assets[i].Volatility, assets[j].Volatility
		default:
			vi, vj = assets[i].TotalValue, assets[j].TotalValue
		}
		if desc {
			return vi > vj
		}
		return vi < vj
	})
}

// buildDashboardData builds the dashboard data structure.
func buildDashboardData(assets []model.AssetPerformance, filter, sortBy string, sortDesc bool) model.PerformanceDashboardData {
	data := model.PerformanceDashboardData{
		Assets:   assets,
		Filter:   filter,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	if len(assets) == 0 {
		return data
	}

	// Find best/worst performers and average returns
	var totalReturn, totalDaily, totalWeekly, totalMonthly float64
	bestYTD := assets[0].YTDChange
	worstYTD := assets[0].YTDChange
	data.BestPerformer = assets[0].Name
	data.WorstPerformer = assets[0].Name

	for _, a := range assets {
		totalReturn += a.YTDChange
		totalDaily += a.DailyChange
		totalWeekly += a.WeeklyChange
		totalMonthly += a.MonthlyChange
		if a.YTDChange > bestYTD {
			bestYTD = a.YTDChange
			data.BestPerformer = a.Name
		}
		if a.YTDChange < worstYTD {
			worstYTD = a.YTDChange
			data.WorstPerformer = a.Name
		}
	}

	n := float64(len(assets))
	data.AverageReturn = totalReturn / n
	data.AverageDailyReturn = totalDaily / n
	data.AverageWeeklyReturn = totalWeekly / n
	data.AverageMonthlyReturn = totalMonthly / n

	return data
}
