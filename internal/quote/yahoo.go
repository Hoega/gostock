package quote

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"
)

type rateCache struct {
	rate      float64
	fetchedAt time.Time
	mu        sync.Mutex
}

var exchangeCache = struct {
	mu    sync.Mutex
	items map[string]*rateCache
}{items: make(map[string]*rateCache)}

// QuoteResult holds the data returned by a Yahoo Finance ISIN lookup.
type QuoteResult struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Sector   string  `json:"sector"`
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

// preferredExchanges maps ISIN country codes (first 2 chars) to Yahoo Finance
// exchange identifiers. When multiple Yahoo results are returned for an ISIN,
// the first result whose exchange appears in this list is selected.
var preferredExchanges = map[string][]string{
	"NL": {"AMS"},
	"FR": {"PAR"},
	"DE": {"GER", "FRA"},
	"IE": {"ISE", "LSE", "AMS"},
	"LU": {"AMS", "PAR"},
	"US": {"NMS", "NYQ", "NGM", "PCX", "NAS", "ASE", "NCM", "OPR"},
	"GB": {"LSE"},
	"CH": {"EBS"},
	"IT": {"MIL"},
	"ES": {"MCE"},
	"BE": {"BRU"},
	"PT": {"LIS"},
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// symbolPattern extracts base ticker from Yahoo symbols like "1GOOGL.MI" -> "GOOGL"
var symbolPattern = regexp.MustCompile(`^[0-9]*([A-Z]+)(?:\.[A-Z]+)?$`)

// LookupISIN resolves an ISIN to a quote via Yahoo Finance.
func LookupISIN(isin string) (*QuoteResult, error) {
	symbols, name, sector, err := searchISIN(isin)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// Expected currency based on ISIN country code.
	expectedCurrency := ""
	isUS := len(isin) >= 2 && isin[:2] == "US"
	if isUS {
		expectedCurrency = "USD"
	}

	// Try each symbol candidate until we find one with the expected currency.
	var lastErr error
	for _, symbol := range symbols {
		price, currency, err := fetchPrice(symbol)
		if err != nil {
			lastErr = err
			continue
		}

		// If we have an expected currency, skip listings that don't match.
		if expectedCurrency != "" && currency != expectedCurrency {
			continue
		}

		return &QuoteResult{
			Name:     name,
			Price:    price,
			Currency: currency,
			Sector:   sector,
		}, nil
	}

	// For US ISINs, try to extract base symbol and fetch directly from US exchange.
	// Yahoo sometimes returns only European listings (e.g., "1GOOGL.MI" for Alphabet).
	if isUS && len(symbols) > 0 {
		for _, symbol := range symbols {
			if matches := symbolPattern.FindStringSubmatch(symbol); len(matches) > 1 {
				baseSymbol := matches[1]
				price, currency, err := fetchPrice(baseSymbol)
				if err == nil && currency == "USD" {
					return &QuoteResult{
						Name:     name,
						Price:    price,
						Currency: currency,
						Sector:   sector,
					}, nil
				}
			}
		}
	}

	// Fallback: return the first symbol's data even if currency doesn't match.
	if len(symbols) > 0 {
		price, currency, err := fetchPrice(symbols[0])
		if err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("chart: %w", lastErr)
			}
			return nil, fmt.Errorf("chart: %w", err)
		}
		return &QuoteResult{
			Name:     name,
			Price:    price,
			Currency: currency,
			Sector:   sector,
		}, nil
	}

	return nil, fmt.Errorf("no valid symbol found for ISIN %s", isin)
}

// searchISIN resolves an ISIN to Yahoo Finance symbol candidates, name, and sector.
// Returns symbols ordered by preference (preferred exchanges first).
func searchISIN(isin string) (symbols []string, name, sector string, err error) {
	u := "https://query2.finance.yahoo.com/v1/finance/search?q=" + url.QueryEscape(isin) + "&quotesCount=10&newsCount=0&enableFuzzyQuery=false"

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("yahoo search returned %d", resp.StatusCode)
	}

	var result struct {
		Quotes []struct {
			Symbol    string `json:"symbol"`
			ShortName string `json:"shortname"`
			Sector    string `json:"sector"`
			Exchange  string `json:"exchange"`
		} `json:"quotes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", "", fmt.Errorf("decode search response: %w", err)
	}

	if len(result.Quotes) == 0 {
		return nil, "", "", fmt.Errorf("no results for ISIN %s", isin)
	}

	// Use first quote for name/sector.
	name = result.Quotes[0].ShortName
	sector = result.Quotes[0].Sector

	// Build symbol list: preferred exchanges first, then the rest.
	seen := make(map[string]bool)
	if len(isin) >= 2 {
		if preferred, ok := preferredExchanges[isin[:2]]; ok {
			for _, ex := range preferred {
				for _, quote := range result.Quotes {
					if quote.Exchange == ex && !seen[quote.Symbol] {
						symbols = append(symbols, quote.Symbol)
						seen[quote.Symbol] = true
					}
				}
			}
		}
	}

	// Add remaining symbols as fallback.
	for _, quote := range result.Quotes {
		if !seen[quote.Symbol] {
			symbols = append(symbols, quote.Symbol)
			seen[quote.Symbol] = true
		}
	}

	return symbols, name, sector, nil
}

// FetchExchangeRate returns the exchange rate from one currency to another
// using Yahoo Finance (e.g. EURUSD=X for EUR to USD).
// Results are cached for 24 hours to avoid repeated HTTP calls.
func FetchExchangeRate(from, to string) (float64, error) {
	key := from + to

	exchangeCache.mu.Lock()
	entry, ok := exchangeCache.items[key]
	if !ok {
		entry = &rateCache{}
		exchangeCache.items[key] = entry
	}
	exchangeCache.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.rate != 0 && time.Since(entry.fetchedAt) < 24*time.Hour {
		return entry.rate, nil
	}

	symbol := from + to + "=X"
	price, _, err := fetchPrice(symbol)
	if err != nil {
		return 0, fmt.Errorf("exchange rate %s→%s: %w", from, to, err)
	}

	entry.rate = price
	entry.fetchedAt = time.Now()
	return price, nil
}

// yahooRangeParams maps a user-facing range key to Yahoo Finance API params.
var yahooRangeParams = map[string][2]string{
	"1m":  {"1mo", "1d"},
	"3m":  {"3mo", "1d"},
	"1y":  {"1y", "1d"},
	"5y":  {"5y", "1mo"},
	"max": {"max", "1mo"},
}

// ResolveISINToSymbol resolves an ISIN to the best Yahoo Finance symbol.
func ResolveISINToSymbol(isin string) (string, error) {
	symbols, _, _, err := searchISIN(isin)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return "", fmt.Errorf("no symbol found for ISIN %s", isin)
	}
	return symbols[0], nil
}

// FetchStockHistory returns historical price data for a Yahoo Finance symbol.
// rangeKey must be one of: 1m, 3m, 1y, 5y, max.
func FetchStockHistory(symbol, rangeKey string) ([]PricePoint, error) {
	params, ok := yahooRangeParams[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range: %s", rangeKey)
	}

	u := "https://query1.finance.yahoo.com/v8/finance/chart/" + url.PathEscape(symbol) +
		"?range=" + params[0] + "&interval=" + params[1]

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo chart returned %d for %s", resp.StatusCode, symbol)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close [](*float64) `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode chart history: %w", err)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data for %s", symbol)
	}

	r := result.Chart.Result[0]
	if len(r.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data for %s", symbol)
	}

	timestamps := r.Timestamp
	closes := r.Indicators.Quote[0].Close

	// Normalize timestamps and deduplicate (keep last value per bucket).
	// For daily intervals, normalize to midnight UTC.
	// For monthly intervals, normalize to the 1st of the month so that
	// stocks from different exchanges align when aggregating portfolio totals.
	interval := params[1]
	bucketMap := make(map[int64]float64)
	var bucketOrder []int64
	for i, ts := range timestamps {
		if i < len(closes) && closes[i] != nil {
			t := time.Unix(ts, 0).UTC()
			var bucketTs int64
			if interval == "1mo" {
				bucketTs = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()
			} else {
				bucketTs = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
			}
			if _, exists := bucketMap[bucketTs]; !exists {
				bucketOrder = append(bucketOrder, bucketTs)
			}
			bucketMap[bucketTs] = *closes[i]
		}
	}

	points := make([]PricePoint, 0, len(bucketOrder))
	for _, ts := range bucketOrder {
		points = append(points, PricePoint{Timestamp: ts, Price: bucketMap[ts]})
	}

	return points, nil
}

// fetchPrice gets the current price and currency for a Yahoo Finance symbol.
func fetchPrice(symbol string) (price float64, currency string, err error) {
	u := "https://query1.finance.yahoo.com/v8/finance/chart/" + url.PathEscape(symbol) + "?range=1d&interval=1d"

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("yahoo chart returned %d", resp.StatusCode)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					Currency           string  `json:"currency"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("decode chart response: %w", err)
	}

	if len(result.Chart.Result) == 0 {
		return 0, "", fmt.Errorf("no chart data for %s", symbol)
	}

	meta := result.Chart.Result[0].Meta
	return meta.RegularMarketPrice, meta.Currency, nil
}
