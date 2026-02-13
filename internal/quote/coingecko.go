package quote

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CryptoQuoteResult holds the data returned by a CoinGecko lookup.
type CryptoQuoteResult struct {
	ID     string  `json:"id"`     // coingecko ID (bitcoin, ethereum)
	Symbol string  `json:"symbol"` // BTC, ETH
	Name   string  `json:"name"`
	Price  float64 `json:"price"` // price in EUR
}

type cryptoPriceCache struct {
	prices    map[string]float64
	fetchedAt time.Time
	mu        sync.Mutex
}

var cryptoCache = &cryptoPriceCache{
	prices: make(map[string]float64),
}

const cryptoCacheDuration = 5 * time.Minute

// cryptoHistEntry holds a cached history response with its own mutex for
// singleflight behavior: concurrent requests for the same key wait for
// the first fetch to complete instead of all hitting the API.
type cryptoHistEntry struct {
	mu        sync.Mutex
	points    []PricePoint
	fetchedAt time.Time
}

var cryptoHistCache = struct {
	mu    sync.Mutex
	items map[string]*cryptoHistEntry
}{items: make(map[string]*cryptoHistEntry)}

// coingeckoSem limits concurrent CoinGecko API requests to avoid 429 rate limiting.
var coingeckoSem = make(chan struct{}, 5)

// SearchCrypto searches for a cryptocurrency by symbol or name and returns the best match.
func SearchCrypto(query string) (*CryptoQuoteResult, error) {
	u := "https://api.coingecko.com/api/v3/search?query=" + url.QueryEscape(query)

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
		return nil, fmt.Errorf("coingecko search returned %d", resp.StatusCode)
	}

	var result struct {
		Coins []struct {
			ID     string `json:"id"`
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"coins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	if len(result.Coins) == 0 {
		return nil, fmt.Errorf("no results for query %s", query)
	}

	// Find exact symbol match first (case insensitive)
	queryUpper := strings.ToUpper(query)
	for _, coin := range result.Coins {
		if strings.ToUpper(coin.Symbol) == queryUpper {
			price, err := FetchCryptoPrice(coin.ID)
			if err != nil {
				return nil, err
			}
			return &CryptoQuoteResult{
				ID:     coin.ID,
				Symbol: strings.ToUpper(coin.Symbol),
				Name:   coin.Name,
				Price:  price,
			}, nil
		}
	}

	// Fallback to first result
	coin := result.Coins[0]
	price, err := FetchCryptoPrice(coin.ID)
	if err != nil {
		return nil, err
	}

	return &CryptoQuoteResult{
		ID:     coin.ID,
		Symbol: strings.ToUpper(coin.Symbol),
		Name:   coin.Name,
		Price:  price,
	}, nil
}

// FetchCryptoPrice fetches the current EUR price for a single cryptocurrency.
func FetchCryptoPrice(coingeckoID string) (float64, error) {
	prices, err := FetchCryptoPricesBatch([]string{coingeckoID})
	if err != nil {
		return 0, err
	}
	price, ok := prices[coingeckoID]
	if !ok {
		return 0, fmt.Errorf("no price found for %s", coingeckoID)
	}
	return price, nil
}

// coingeckoDaysParam maps a user-facing range key to CoinGecko days param.
var coingeckoDaysParam = map[string]string{
	"1m":  "30",
	"3m":  "90",
	"1y":  "365",
	"5y":  "1825",
	"max": "max",
}

const cryptoHistCacheDuration = 10 * time.Minute

// FetchCryptoHistory returns historical price data for a cryptocurrency from CoinGecko.
// rangeKey must be one of: 1m, 3m, 1y, 5y, max.
// Uses singleflight caching: concurrent requests for the same ID+range share one API call.
func FetchCryptoHistory(coingeckoID, rangeKey string) ([]PricePoint, error) {
	days, ok := coingeckoDaysParam[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range: %s", rangeKey)
	}

	cacheKey := coingeckoID + ":" + rangeKey

	// Get or create the cache entry (singleflight pattern)
	cryptoHistCache.mu.Lock()
	entry, exists := cryptoHistCache.items[cacheKey]
	if !exists {
		entry = &cryptoHistEntry{}
		cryptoHistCache.items[cacheKey] = entry
	}
	cryptoHistCache.mu.Unlock()

	// Lock the entry: concurrent requests for the same key wait here
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if cached data is still valid
	if entry.points != nil && time.Since(entry.fetchedAt) < cryptoHistCacheDuration {
		return entry.points, nil
	}

	// Acquire semaphore to limit concurrent API calls
	coingeckoSem <- struct{}{}
	defer func() { <-coingeckoSem }()

	u := "https://api.coingecko.com/api/v3/coins/" + url.PathEscape(coingeckoID) +
		"/market_chart?vs_currency=eur&days=" + days

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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("coingecko rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko market_chart returned %d for %s", resp.StatusCode, coingeckoID)
	}

	var result struct {
		Prices [][]float64 `json:"prices"` // [[timestamp_ms, price], ...]
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode market_chart response: %w", err)
	}

	// Normalize timestamps to midnight UTC and deduplicate by day
	// (keep last value per day). CoinGecko returns hourly data for
	// ranges <= 90 days, so this collapses to one point per day.
	dayMap := make(map[int64]float64)
	var dayOrder []int64
	for _, pair := range result.Prices {
		if len(pair) == 2 {
			t := time.UnixMilli(int64(pair[0])).UTC()
			dayTs := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
			if _, exists := dayMap[dayTs]; !exists {
				dayOrder = append(dayOrder, dayTs)
			}
			dayMap[dayTs] = pair[1]
		}
	}

	points := make([]PricePoint, 0, len(dayOrder))
	for _, ts := range dayOrder {
		points = append(points, PricePoint{Timestamp: ts, Price: dayMap[ts]})
	}

	// Cache the result
	entry.points = points
	entry.fetchedAt = time.Now()

	return points, nil
}

// FetchCryptoPricesBatch fetches EUR prices for multiple cryptocurrencies in a single request.
// Results are cached for 5 minutes.
func FetchCryptoPricesBatch(ids []string) (map[string]float64, error) {
	if len(ids) == 0 {
		return make(map[string]float64), nil
	}

	cryptoCache.mu.Lock()
	defer cryptoCache.mu.Unlock()

	// Check cache for all IDs
	allCached := true
	cacheValid := time.Since(cryptoCache.fetchedAt) < cryptoCacheDuration
	if cacheValid {
		for _, id := range ids {
			if _, ok := cryptoCache.prices[id]; !ok {
				allCached = false
				break
			}
		}
		if allCached {
			result := make(map[string]float64)
			for _, id := range ids {
				result[id] = cryptoCache.prices[id]
			}
			return result, nil
		}
	}

	// Fetch from API
	idsParam := strings.Join(ids, ",")
	u := "https://api.coingecko.com/api/v3/simple/price?ids=" + url.QueryEscape(idsParam) + "&vs_currencies=eur"

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
		return nil, fmt.Errorf("coingecko price returned %d", resp.StatusCode)
	}

	var result map[string]struct {
		EUR float64 `json:"eur"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode price response: %w", err)
	}

	// Update cache
	cryptoCache.fetchedAt = time.Now()
	prices := make(map[string]float64)
	for id, data := range result {
		cryptoCache.prices[id] = data.EUR
		prices[id] = data.EUR
	}

	return prices, nil
}
