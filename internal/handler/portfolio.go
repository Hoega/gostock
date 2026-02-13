package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
	"github.com/Hoega/gostock/internal/quote"
)

// PortfolioPageData holds all data needed to render the portfolio page.
type PortfolioPageData struct {
	ActiveTab    string
	StockSummary model.PortfolioSummary
	CryptoSummary model.CryptoSummary
}

type PortfolioHandler struct {
	templates *template.Template
	store     persistence.Store
}

func NewPortfolioHandler(templates *template.Template, store persistence.Store) *PortfolioHandler {
	return &PortfolioHandler{templates: templates, store: store}
}

// ShowPortfolio renders the portfolio page with all positions.
func (h *PortfolioHandler) ShowPortfolio(w http.ResponseWriter, r *http.Request) {
	activeTab := r.URL.Query().Get("tab")
	if activeTab != "crypto" {
		activeTab = "stocks"
	}

	data := PortfolioPageData{ActiveTab: activeTab}

	if activeTab == "stocks" {
		summary, err := h.loadStockSummary()
		if err != nil {
			log.Printf("Failed to load stock positions: %v", err)
			summary = model.ComputePortfolioSummary(nil, nil)
		}
		data.StockSummary = summary
	} else {
		summary, err := h.loadCryptoSummary()
		if err != nil {
			log.Printf("Failed to load crypto positions: %v", err)
			summary = model.ComputeCryptoSummary(nil)
		}
		data.CryptoSummary = summary
	}

	// If HTMX request, render only the container partial
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "portfolio-container.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.ExecuteTemplate(w, "portfolio.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AddPosition handles POST /portfolio/positions.
func (h *PortfolioHandler) AddPosition(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	pos := h.parsePositionForm(r)
	if err := h.store.SavePosition(pos); err != nil {
		log.Printf("Failed to save position: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w)
}

// UpdatePosition handles PUT /portfolio/positions/{id}.
func (h *PortfolioHandler) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	pos := h.parsePositionForm(r)
	pos.ID = id
	if err := h.store.SavePosition(pos); err != nil {
		log.Printf("Failed to update position: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w)
}

// DeletePosition handles DELETE /portfolio/positions/{id}.
func (h *PortfolioHandler) DeletePosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := h.store.DeletePosition(id); err != nil {
		log.Printf("Failed to delete position: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w)
}

// LookupQuote handles GET /portfolio/quote?isin={ISIN} and returns JSON.
func (h *PortfolioHandler) LookupQuote(w http.ResponseWriter, r *http.Request) {
	isin := r.URL.Query().Get("isin")
	if len(isin) != 12 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ISIN invalide (12 caractères attendus)"})
		return
	}

	result, err := quote.LookupISIN(isin)
	if err != nil {
		log.Printf("Quote lookup failed for %s: %v", isin, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Impossible de trouver ce titre"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// AddCryptoPosition handles POST /portfolio/crypto/positions.
func (h *PortfolioHandler) AddCryptoPosition(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	pos := h.parseCryptoPositionForm(r)
	if err := h.store.SaveCryptoPosition(pos); err != nil {
		log.Printf("Failed to save crypto position: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w)
}

// UpdateCryptoPosition handles PUT /portfolio/crypto/positions/{id}.
func (h *PortfolioHandler) UpdateCryptoPosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	pos := h.parseCryptoPositionForm(r)
	pos.ID = id
	if err := h.store.SaveCryptoPosition(pos); err != nil {
		log.Printf("Failed to update crypto position: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w)
}

// DeleteCryptoPosition handles DELETE /portfolio/crypto/positions/{id}.
func (h *PortfolioHandler) DeleteCryptoPosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteCryptoPosition(id); err != nil {
		log.Printf("Failed to delete crypto position: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w)
}

// LookupCryptoQuote handles GET /portfolio/crypto/quote?symbol={SYMBOL} and returns JSON.
func (h *PortfolioHandler) LookupCryptoQuote(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Symbole requis"})
		return
	}

	result, err := quote.SearchCrypto(symbol)
	if err != nil {
		log.Printf("Crypto quote lookup failed for %s: %v", symbol, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Impossible de trouver cette crypto"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// StockHistory handles GET /portfolio/history?isin={ISIN}&range={1m|3m|1y|5y|max}.
func (h *PortfolioHandler) StockHistory(w http.ResponseWriter, r *http.Request) {
	isin := r.URL.Query().Get("isin")
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1y"
	}

	symbol, err := quote.ResolveISINToSymbol(isin)
	if err != nil {
		log.Printf("Resolve ISIN %s failed: %v", isin, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Symbole introuvable"})
		return
	}

	points, err := quote.FetchStockHistory(symbol, rangeKey)
	if err != nil {
		log.Printf("Stock history failed for %s (%s): %v", symbol, rangeKey, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Impossible de charger l'historique"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

// CryptoHistory handles GET /portfolio/crypto/history?id={coingecko_id}&range={1m|3m|1y|5y|max}.
func (h *PortfolioHandler) CryptoHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1y"
	}

	points, err := quote.FetchCryptoHistory(id, rangeKey)
	if err != nil {
		log.Printf("Crypto history failed for %s (%s): %v", id, rangeKey, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Impossible de charger l'historique"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

// TotalHistory handles GET /portfolio/history/total?tab={stocks|crypto}&range={range}.
// Returns the aggregated portfolio value over time.
func (h *PortfolioHandler) TotalHistory(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1y"
	}

	w.Header().Set("Content-Type", "application/json")

	if tab == "crypto" {
		h.totalCryptoHistory(w, rangeKey)
	} else {
		h.totalStockHistory(w, rangeKey)
	}
}

func (h *PortfolioHandler) totalStockHistory(w http.ResponseWriter, rangeKey string) {
	positions, err := h.store.LoadPositions()
	if err != nil {
		log.Printf("Load positions for total history: %v", err)
		json.NewEncoder(w).Encode([]quote.PricePoint{})
		return
	}

	if len(positions) == 0 {
		json.NewEncoder(w).Encode([]quote.PricePoint{})
		return
	}

	// Fetch history for each position in parallel
	type posHistory struct {
		points   []quote.PricePoint
		quantity float64
	}
	results := make([]posHistory, len(positions))
	var wg sync.WaitGroup
	for i, pos := range positions {
		wg.Add(1)
		go func(idx int, p persistence.StockPosition) {
			defer wg.Done()
			symbol, err := quote.ResolveISINToSymbol(p.ISIN)
			if err != nil {
				log.Printf("Resolve %s: %v", p.ISIN, err)
				return
			}
			pts, err := quote.FetchStockHistory(symbol, rangeKey)
			if err != nil {
				log.Printf("History %s: %v", symbol, err)
				return
			}
			results[idx] = posHistory{points: pts, quantity: p.Quantity}
		}(i, pos)
	}
	wg.Wait()

	// Aggregate: sum values per timestamp
	tsMap := make(map[int64]float64)
	for _, ph := range results {
		for _, pt := range ph.points {
			tsMap[pt.Timestamp] += pt.Price * ph.quantity
		}
	}

	// Sort by timestamp
	points := make([]quote.PricePoint, 0, len(tsMap))
	for ts, val := range tsMap {
		points = append(points, quote.PricePoint{Timestamp: ts, Price: val})
	}
	sortPricePoints(points)

	json.NewEncoder(w).Encode(points)
}

func (h *PortfolioHandler) totalCryptoHistory(w http.ResponseWriter, rangeKey string) {
	positions, err := h.store.LoadCryptoPositions()
	if err != nil {
		log.Printf("Load crypto positions for total history: %v", err)
		json.NewEncoder(w).Encode([]quote.PricePoint{})
		return
	}

	if len(positions) == 0 {
		json.NewEncoder(w).Encode([]quote.PricePoint{})
		return
	}

	type posHistory struct {
		points   []quote.PricePoint
		quantity float64
	}
	results := make([]posHistory, len(positions))
	var wg sync.WaitGroup
	for i, pos := range positions {
		wg.Add(1)
		go func(idx int, p persistence.CryptoPosition) {
			defer wg.Done()
			pts, err := quote.FetchCryptoHistory(p.CoingeckoID, rangeKey)
			if err != nil {
				log.Printf("Crypto history %s: %v", p.CoingeckoID, err)
				return
			}
			results[idx] = posHistory{points: pts, quantity: p.Quantity}
		}(i, pos)
	}
	wg.Wait()

	tsMap := make(map[int64]float64)
	for _, ph := range results {
		for _, pt := range ph.points {
			tsMap[pt.Timestamp] += pt.Price * ph.quantity
		}
	}

	points := make([]quote.PricePoint, 0, len(tsMap))
	for ts, val := range tsMap {
		points = append(points, quote.PricePoint{Timestamp: ts, Price: val})
	}
	sortPricePoints(points)

	json.NewEncoder(w).Encode(points)
}

func sortPricePoints(points []quote.PricePoint) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})
}

func (h *PortfolioHandler) parseCryptoPositionForm(r *http.Request) *persistence.CryptoPosition {
	return &persistence.CryptoPosition{
		Symbol:        r.FormValue("symbol"),
		CoingeckoID:   r.FormValue("coingecko_id"),
		Name:          r.FormValue("name"),
		Wallet:        r.FormValue("wallet"),
		Quantity:      pParseFloat(r.FormValue("quantity"), 0),
		PurchasePrice: pParseFloat(r.FormValue("purchase_price"), 0),
		CurrentPrice:  pParseFloat(r.FormValue("current_price"), 0),
		PurchaseFees:  pParseFloat(r.FormValue("purchase_fees"), 0),
	}
}

func (h *PortfolioHandler) parsePositionForm(r *http.Request) *persistence.StockPosition {
	currency := r.FormValue("currency")
	if currency == "" {
		currency = "EUR"
	}
	return &persistence.StockPosition{
		Name:          r.FormValue("name"),
		ISIN:          r.FormValue("isin"),
		Broker:        r.FormValue("broker"),
		Quantity:      pParseFloat(r.FormValue("quantity"), 0),
		PurchasePrice: pParseFloat(r.FormValue("purchase_price"), 0),
		CurrentPrice:  pParseFloat(r.FormValue("current_price"), 0),
		PurchaseFees:  pParseFloat(r.FormValue("purchase_fees"), 0),
		Currency:      currency,
		Sector:        r.FormValue("sector"),
	}
}

func (h *PortfolioHandler) loadStockSummary() (model.PortfolioSummary, error) {
	dbPositions, err := h.store.LoadPositions()
	if err != nil {
		return model.PortfolioSummary{}, err
	}

	positions := make([]model.StockPosition, len(dbPositions))
	needUSD := false
	for i, p := range dbPositions {
		positions[i] = model.StockPosition{
			ID:            p.ID,
			Name:          p.Name,
			ISIN:          p.ISIN,
			Broker:        p.Broker,
			Quantity:      p.Quantity,
			PurchasePrice: p.PurchasePrice,
			CurrentPrice:  p.CurrentPrice,
			PurchaseFees:  p.PurchaseFees,
			Currency:      p.Currency,
			Sector:        p.Sector,
		}
		if p.Currency == "USD" {
			needUSD = true
		}
	}

	rates := map[string]float64{"EUR": 1.0}
	if needUSD {
		eurusd, err := quote.FetchExchangeRate("EUR", "USD")
		if err != nil {
			log.Printf("Failed to fetch EUR/USD rate: %v", err)
			eurusd = 1.0 // fallback: no conversion
		}
		if eurusd > 0 {
			rates["USD"] = 1.0 / eurusd
		}
	}

	return model.ComputePortfolioSummary(positions, rates), nil
}

func (h *PortfolioHandler) loadCryptoSummary() (model.CryptoSummary, error) {
	dbPositions, err := h.store.LoadCryptoPositions()
	if err != nil {
		return model.CryptoSummary{}, err
	}

	positions := make([]model.CryptoPosition, len(dbPositions))
	var coingeckoIDs []string
	for i, p := range dbPositions {
		positions[i] = model.CryptoPosition{
			ID:            p.ID,
			Symbol:        p.Symbol,
			CoingeckoID:   p.CoingeckoID,
			Name:          p.Name,
			Wallet:        p.Wallet,
			Quantity:      p.Quantity,
			PurchasePrice: p.PurchasePrice,
			CurrentPrice:  p.CurrentPrice,
			PurchaseFees:  p.PurchaseFees,
		}
		coingeckoIDs = append(coingeckoIDs, p.CoingeckoID)
	}

	// Refresh prices from CoinGecko (batch request)
	if len(coingeckoIDs) > 0 {
		prices, err := quote.FetchCryptoPricesBatch(coingeckoIDs)
		if err != nil {
			log.Printf("Failed to fetch crypto prices: %v", err)
		} else {
			for i := range positions {
				if price, ok := prices[positions[i].CoingeckoID]; ok {
					positions[i].CurrentPrice = price
				}
			}
		}
	}

	return model.ComputeCryptoSummary(positions), nil
}

func (h *PortfolioHandler) renderStockPartials(w http.ResponseWriter) {
	summary, err := h.loadStockSummary()
	if err != nil {
		log.Printf("Failed to load positions: %v", err)
		summary = model.ComputePortfolioSummary(nil, nil)
	}

	if err := h.templates.ExecuteTemplate(w, "portfolio-table.html", summary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "portfolio-charts.html", summary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *PortfolioHandler) renderCryptoPartials(w http.ResponseWriter) {
	summary, err := h.loadCryptoSummary()
	if err != nil {
		log.Printf("Failed to load crypto positions: %v", err)
		summary = model.ComputeCryptoSummary(nil)
	}

	if err := h.templates.ExecuteTemplate(w, "crypto-table.html", summary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "crypto-charts.html", summary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pParseFloat(s string, fallback float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}
