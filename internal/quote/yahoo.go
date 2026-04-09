package quote

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
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

// yahooCrumb holds a cached crumb + authenticated HTTP client for Yahoo Finance
// endpoints that require authentication (e.g. quoteSummary v10).
var yahooCrumb struct {
	mu        sync.Mutex
	crumb     string
	client    *http.Client
	fetchedAt time.Time
}

// getYahooCrumb returns a crumb string and an HTTP client with valid cookies.
// The crumb is cached for 1 hour. Thread-safe.
func getYahooCrumb() (string, *http.Client, error) {
	yahooCrumb.mu.Lock()
	defer yahooCrumb.mu.Unlock()

	if yahooCrumb.crumb != "" && time.Since(yahooCrumb.fetchedAt) < 1*time.Hour {
		return yahooCrumb.crumb, yahooCrumb.client, nil
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}

	// Step 1: hit a Yahoo page to get cookies
	req, err := http.NewRequest("GET", "https://fc.yahoo.com", nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("yahoo cookie fetch: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Step 2: fetch the crumb using the cookies
	req, err = http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err = client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("yahoo crumb fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("yahoo crumb returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read crumb: %w", err)
	}

	crumb := string(body)
	if crumb == "" {
		return "", nil, fmt.Errorf("empty crumb from Yahoo")
	}

	yahooCrumb.crumb = crumb
	yahooCrumb.client = client
	yahooCrumb.fetchedAt = time.Now()

	return crumb, client, nil
}

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

	isUS := len(isin) >= 2 && isin[:2] == "US"

	// For US ISINs, try the resolved base symbol first (1 API call instead of N).
	// This avoids rate limiting when Yahoo search only returns European listings.
	if isUS {
		baseSymbol := resolveUSBaseSymbol(symbols)
		price, currency, err := fetchPrice(baseSymbol)
		if err == nil && currency == "USD" {
			return &QuoteResult{
				Name:     name,
				Price:    price,
				Currency: currency,
				Sector:   sector,
			}, nil
		}
		// If the base symbol is too long (e.g., "GOOGLCO" from "GOOGLCO.CL"),
		// try shorter prefixes to find the real US ticker.
		if len(baseSymbol) > 5 {
			if _, price, err := tryUSTickerPrefixes(baseSymbol); err == nil {
				return &QuoteResult{
					Name:     name,
					Price:    price,
					Currency: "USD",
					Sector:   sector,
				}, nil
			}
		}
	}

	// Try each symbol candidate until we find one with the expected currency.
	expectedCurrency := ""
	if isUS {
		expectedCurrency = "USD"
	}

	var lastErr error
	for _, symbol := range symbols {
		price, currency, err := fetchPrice(symbol)
		if err != nil {
			lastErr = err
			continue
		}

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

// resolveUSBaseSymbol extracts the US base ticker from Yahoo search results
// without making any API calls. For example, if Yahoo only returns European
// listings like "1GOOGL.MI", this extracts "GOOGL".
func resolveUSBaseSymbol(symbols []string) string {
	if len(symbols) == 0 {
		return ""
	}

	// Pass 1: Look for a short undotted symbol (1-5 chars, all uppercase).
	// US tickers are almost always ≤5 chars (e.g., GOOGL, AMZN, AAPL).
	// This avoids treating concatenated ticker+exchange like "GOOGLCO" as a clean US ticker.
	for _, s := range symbols {
		if symbolPattern.MatchString(s) {
			m := symbolPattern.FindStringSubmatch(s)
			if m[1] == s && len(s) <= 5 {
				return s
			}
		}
	}

	// Pass 2: Extract base ticker from dotted symbols (e.g., "GOOGL.MI" → "GOOGL").
	// Only accept bases ≤5 chars to avoid concatenated ticker+exchange like "GOOGLCO" from "GOOGLCO.CL".
	for _, symbol := range symbols {
		if m := symbolPattern.FindStringSubmatch(symbol); len(m) > 1 && m[1] != symbol && len(m[1]) <= 5 {
			return m[1]
		}
	}

	// Pass 3: Extract from numeric-prefixed symbols (e.g., "1GOOGL" → "GOOGL").
	for _, symbol := range symbols {
		if m := symbolPattern.FindStringSubmatch(symbol); len(m) > 1 && len(m[1]) <= 5 {
			return m[1]
		}
	}

	return symbols[0]
}

// tryUSTickerPrefixes tries progressively shorter prefixes of a long base symbol
// to find the real US ticker. For example, "GOOGLCO" → tries "GOOGL", "GOOG", "GOO".
// This handles cases where Yahoo search only returns non-US listings with concatenated
// ticker+exchange suffixes (e.g., "GOOGLCO.CL" for Alphabet on the Colombian exchange).
func tryUSTickerPrefixes(base string) (string, float64, error) {
	maxLen := min(len(base)-1, 5)
	for l := maxLen; l >= 2; l-- {
		candidate := base[:l]
		price, currency, err := fetchPrice(candidate)
		if err == nil && currency == "USD" {
			return candidate, price, nil
		}
	}
	return "", 0, fmt.Errorf("no valid US ticker found from prefix %s", base)
}

// ResolveISINToSymbol resolves an ISIN to the best Yahoo Finance symbol.
// For US ISINs, it extracts the base ticker without extra API calls to avoid
// rate limiting when called concurrently for multiple positions.
func ResolveISINToSymbol(isin string) (string, error) {
	symbols, _, _, err := searchISIN(isin)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return "", fmt.Errorf("no symbol found for ISIN %s", isin)
	}

	isUS := len(isin) >= 2 && isin[:2] == "US"
	if isUS {
		base := resolveUSBaseSymbol(symbols)
		if len(base) <= 5 {
			return base, nil
		}
		// Base is too long (e.g., "GOOGLCO" from "GOOGLCO.CL").
		// Try shorter prefixes via fetchPrice to find the real US ticker.
		if symbol, _, err := tryUSTickerPrefixes(base); err == nil {
			return symbol, nil
		}
		return base, nil
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

// StockIndicators holds fundamental financial indicators for a stock.
type StockIndicators struct {
	PER              float64 `json:"per"`
	ForwardPER       float64 `json:"forwardPer"`
	PEG              float64 `json:"peg"`
	EPS              float64 `json:"eps"`
	DividendYield    float64 `json:"dividendYield"`
	FiftyTwoWeekHigh float64 `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow  float64 `json:"fiftyTwoWeekLow"`
	MarketCap        float64 `json:"marketCap"`
	Beta             float64 `json:"beta"`
}

// FetchQuoteSummaryFull retrieves fundamental indicators for a Yahoo Finance symbol
// using the quoteSummary endpoint (summaryDetail + defaultKeyStatistics).
// This endpoint requires authentication (crumb + cookie).
func FetchQuoteSummaryFull(symbol string) (*StockIndicators, error) {
	crumb, client, err := getYahooCrumb()
	if err != nil {
		return nil, fmt.Errorf("yahoo auth: %w", err)
	}

	u := "https://query2.finance.yahoo.com/v10/finance/quoteSummary/" + url.PathEscape(symbol) +
		"?modules=summaryDetail,defaultKeyStatistics&crumb=" + url.QueryEscape(crumb)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Invalidate crumb cache on auth failure so next call retries
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			yahooCrumb.mu.Lock()
			yahooCrumb.crumb = ""
			yahooCrumb.mu.Unlock()
		}
		return nil, fmt.Errorf("quoteSummary returned %d for %s", resp.StatusCode, symbol)
	}

	type yahooVal struct {
		Raw float64 `json:"raw"`
	}
	var result struct {
		QuoteSummary struct {
			Result []struct {
				SummaryDetail struct {
					TrailingPE       yahooVal `json:"trailingPE"`
					DividendYield    yahooVal `json:"dividendYield"`
					FiftyTwoWeekHigh yahooVal `json:"fiftyTwoWeekHigh"`
					FiftyTwoWeekLow  yahooVal `json:"fiftyTwoWeekLow"`
					MarketCap        yahooVal `json:"marketCap"`
				} `json:"summaryDetail"`
				DefaultKeyStatistics struct {
					ForwardPE   yahooVal `json:"forwardPE"`
					PegRatio    yahooVal `json:"pegRatio"`
					Beta        yahooVal `json:"beta"`
					TrailingEps yahooVal `json:"trailingEps"`
					MarketCap   yahooVal `json:"enterpriseValue"`
				} `json:"defaultKeyStatistics"`
			} `json:"result"`
		} `json:"quoteSummary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode quoteSummary: %w", err)
	}

	if len(result.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("no quoteSummary data for %s", symbol)
	}

	r := result.QuoteSummary.Result[0]
	ind := &StockIndicators{
		PER:              r.SummaryDetail.TrailingPE.Raw,
		ForwardPER:       r.DefaultKeyStatistics.ForwardPE.Raw,
		PEG:              r.DefaultKeyStatistics.PegRatio.Raw,
		EPS:              r.DefaultKeyStatistics.TrailingEps.Raw,
		DividendYield:    r.SummaryDetail.DividendYield.Raw * 100,
		FiftyTwoWeekHigh: r.SummaryDetail.FiftyTwoWeekHigh.Raw,
		FiftyTwoWeekLow:  r.SummaryDetail.FiftyTwoWeekLow.Raw,
		MarketCap:        r.SummaryDetail.MarketCap.Raw,
		Beta:             r.DefaultKeyStatistics.Beta.Raw,
	}

	// Use enterpriseValue as fallback for marketCap if summaryDetail didn't have it
	if ind.MarketCap == 0 {
		ind.MarketCap = r.DefaultKeyStatistics.MarketCap.Raw
	}

	return ind, nil
}

// ComputePerformance returns price performance (%) at 1W, 1M, 3M, 6M, 1Y horizons.
// It fetches 1-year history and finds the closest data point to each target date.
func ComputePerformance(symbol string, currentPrice float64) (p1w, p1m, p3m, p6m, p1y float64, err error) {
	points, err := FetchStockHistory(symbol, "1y")
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	if len(points) == 0 {
		return 0, 0, 0, 0, 0, fmt.Errorf("no history for %s", symbol)
	}

	now := time.Now()
	horizons := []time.Duration{
		7 * 24 * time.Hour,
		30 * 24 * time.Hour,
		90 * 24 * time.Hour,
		180 * 24 * time.Hour,
		365 * 24 * time.Hour,
	}

	results := make([]float64, len(horizons))
	for i, h := range horizons {
		targetMs := now.Add(-h).UnixMilli()
		// Find closest point
		bestIdx := 0
		bestDist := int64(1<<62 - 1)
		for j, pt := range points {
			dist := pt.Timestamp - targetMs
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}
		oldPrice := points[bestIdx].Price
		if oldPrice > 0 {
			results[i] = (currentPrice - oldPrice) / oldPrice * 100
		}
	}

	return results[0], results[1], results[2], results[3], results[4], nil
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
