package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
	"github.com/Hoega/gostock/internal/quote"
)

// PortfolioPageData holds all data needed to render the portfolio page.
type PortfolioPageData struct {
	ActiveTab          string
	StockSummary       model.PortfolioSummary
	CryptoSummary      model.CryptoSummary
	CashSummary        model.CashSummary
	GlobalSummary      model.GlobalSummary
	RealEstateSummary  model.RealEstateSummary
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
	if activeTab != "stocks" && activeTab != "crypto" && activeTab != "cash" && activeTab != "immobilier" {
		activeTab = "overview"
	}

	data := PortfolioPageData{ActiveTab: activeTab}

	switch activeTab {
	case "overview":
		data.GlobalSummary = h.loadGlobalSummary()
	case "crypto":
		summary, err := h.loadCryptoSummary()
		if err != nil {
			log.Printf("Failed to load crypto positions: %v", err)
			summary = model.ComputeCryptoSummary(nil)
		}
		data.CryptoSummary = summary
	case "cash":
		summary, err := h.loadCashSummary()
		if err != nil {
			log.Printf("Failed to load cash positions: %v", err)
			summary = model.ComputeCashSummary(nil)
		}
		data.CashSummary = summary
	case "immobilier":
		// Parse optional year/month for date slider
		atYear := 0
		atMonth := 0
		if y := r.URL.Query().Get("year"); y != "" {
			atYear, _ = strconv.Atoi(y)
		}
		if m := r.URL.Query().Get("month"); m != "" {
			atMonth, _ = strconv.Atoi(m)
		}
		data.RealEstateSummary = h.loadRealEstateSummaryAt(atYear, atMonth)
	default:
		summary, err := h.loadStockSummary()
		if err != nil {
			log.Printf("Failed to load stock positions: %v", err)
			summary = model.ComputePortfolioSummary(nil, nil)
		}
		data.StockSummary = summary
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

	// Refresh current prices from Yahoo Finance (like crypto does from CoinGecko).
	var wg sync.WaitGroup
	for i := range positions {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			isin := positions[idx].ISIN
			if isin == "" {
				return
			}
			result, err := quote.LookupISIN(isin)
			if err != nil {
				log.Printf("Refresh price for %s failed: %v", isin, err)
				return
			}
			positions[idx].CurrentPrice = result.Price
		}(i)
	}
	wg.Wait()

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

// AddCashPosition handles POST /portfolio/cash/positions.
func (h *PortfolioHandler) AddCashPosition(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	pos := h.parseCashPositionForm(r)
	if err := h.store.SaveCashPosition(pos); err != nil {
		log.Printf("Failed to save cash position: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	h.renderCashPartials(w)
}

// UpdateCashPosition handles PUT /portfolio/cash/positions/{id}.
func (h *PortfolioHandler) UpdateCashPosition(w http.ResponseWriter, r *http.Request) {
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

	pos := h.parseCashPositionForm(r)
	pos.ID = id
	if err := h.store.SaveCashPosition(pos); err != nil {
		log.Printf("Failed to update cash position: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	h.renderCashPartials(w)
}

// DeleteCashPosition handles DELETE /portfolio/cash/positions/{id}.
func (h *PortfolioHandler) DeleteCashPosition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteCashPosition(id); err != nil {
		log.Printf("Failed to delete cash position: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderCashPartials(w)
}

func (h *PortfolioHandler) parseCashPositionForm(r *http.Request) *persistence.CashPosition {
	return &persistence.CashPosition{
		BankName:     r.FormValue("bank_name"),
		Amount:       pParseFloat(r.FormValue("amount"), 0),
		AccountType:  r.FormValue("account_type"),
		InterestRate: pParseFloat(r.FormValue("interest_rate"), 0),
	}
}

func (h *PortfolioHandler) loadCashSummary() (model.CashSummary, error) {
	dbPositions, err := h.store.LoadCashPositions()
	if err != nil {
		return model.CashSummary{}, err
	}

	positions := make([]model.CashPosition, len(dbPositions))
	for i, p := range dbPositions {
		positions[i] = model.CashPosition{
			ID:           p.ID,
			BankName:     p.BankName,
			Amount:       p.Amount,
			AccountType:  p.AccountType,
			InterestRate: p.InterestRate,
		}
	}

	return model.ComputeCashSummary(positions), nil
}

func (h *PortfolioHandler) loadGlobalSummary() model.GlobalSummary {
	var stockSummary model.PortfolioSummary
	var cryptoSummary model.CryptoSummary
	var cashSummary model.CashSummary
	var realEstateSummary model.RealEstateSummary

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		s, err := h.loadStockSummary()
		if err != nil {
			log.Printf("Failed to load stock positions for overview: %v", err)
			s = model.ComputePortfolioSummary(nil, nil)
		}
		stockSummary = s
	}()

	go func() {
		defer wg.Done()
		s, err := h.loadCryptoSummary()
		if err != nil {
			log.Printf("Failed to load crypto positions for overview: %v", err)
			s = model.ComputeCryptoSummary(nil)
		}
		cryptoSummary = s
	}()

	go func() {
		defer wg.Done()
		s, err := h.loadCashSummary()
		if err != nil {
			log.Printf("Failed to load cash positions for overview: %v", err)
			s = model.ComputeCashSummary(nil)
		}
		cashSummary = s
	}()

	go func() {
		defer wg.Done()
		realEstateSummary = h.loadRealEstateSummaryAt(0, 0)
	}()

	wg.Wait()
	return model.ComputeGlobalSummary(stockSummary, cryptoSummary, cashSummary, realEstateSummary)
}

func (h *PortfolioHandler) loadRealEstateSummaryAt(reqYear, reqMonth int) model.RealEstateSummary {
	inputs, err := h.store.Load()
	if err != nil {
		log.Printf("Failed to load form inputs for real estate: %v", err)
		return model.RealEstateSummary{}
	}

	now := time.Now()
	atYear := reqYear
	atMonth := reqMonth
	isSliderRequest := reqYear > 0 // Explicit date = slider request, force simulation
	if atYear == 0 {
		atYear = now.Year()
		atMonth = int(now.Month())
	}

	var summary model.RealEstateSummary
	summary.AtYear = atYear
	summary.AtMonth = atMonth

	// Track slider bounds (using year*12 + (month-1) for proper decoding)
	sliderMin := atYear*12 + (atMonth - 1)
	sliderMax := atYear*12 + (atMonth - 1)

	// Current property (bien actuel)
	if inputs.CurrentSalePrice > 0 && inputs.CurrentLoanLines != "" {
		var loanLines []model.LoanLine
		if err := json.Unmarshal([]byte(inputs.CurrentLoanLines), &loanLines); err != nil {
			log.Printf("Failed to parse current loan lines: %v", err)
		} else {
			prop := model.RealEstateProperty{
				Label:         "Bien actuel",
				PropertyValue: inputs.CurrentSalePrice,
			}
			// Fallback start date from global inputs
			fallbackStartYear := inputs.CurrentLoanStartYear
			fallbackStartMonth := inputs.CurrentLoanStartMonth
			if fallbackStartYear == 0 {
				fallbackStartYear = atYear
				fallbackStartMonth = atMonth
			}

			for i := range loanLines {
				line := &loanLines[i]
				if line.OriginalAmount <= 0 {
					continue
				}
				// Use per-line start date if set, otherwise fallback to global start
				lineStartYear := line.StartYear
				lineStartMonth := line.StartMonth
				if lineStartYear == 0 {
					lineStartYear = fallbackStartYear
					lineStartMonth = fallbackStartMonth
				}
				// Set the start date on the line for the calculator
				line.StartYear = lineStartYear
				line.StartMonth = lineStartMonth

				// For slider requests, clear Balance to force simulation from OriginalAmount
				lineCopy := *line
				if isSliderRequest {
					lineCopy.Balance = 0
				}
				remaining, amortized, monthly, monthlyIns := calculator.ComputeLoanRemainingBalance(lineCopy, atYear, atMonth)
				var progressPct float64
				if line.OriginalAmount > 0 {
					progressPct = amortized / line.OriginalAmount * 100
				}

				// Compute end date: start + deferral + amortization duration
				totalMonths := line.DurationYears*12 + line.DeferralMonths
				if line.DurationYears <= 0 && len(line.Tiers) > 0 {
					lastEnd := 0
					for _, t := range line.Tiers {
						if t.EndMonth > lastEnd {
							lastEnd = t.EndMonth
						}
					}
					totalMonths = lastEnd + line.DeferralMonths
				}
				endMonth := lineStartMonth + (totalMonths % 12)
				endYear := lineStartYear + (totalMonths / 12)
				if endMonth > 12 {
					endMonth -= 12
					endYear++
				}

				// Track slider bounds (using year*12 + (month-1) for proper decoding)
				startAbs := lineStartYear*12 + (lineStartMonth - 1)
				endAbs := endYear*12 + (endMonth - 1)
				if startAbs < sliderMin {
					sliderMin = startAbs
				}
				if endAbs > sliderMax {
					sliderMax = endAbs
				}

				prop.Loans = append(prop.Loans, model.RealEstateLoan{
					Label:            line.Label,
					OriginalAmount:   line.OriginalAmount,
					RemainingBalance: remaining,
					AmortizedCapital: amortized,
					Rate:             line.Rate,
					MonthlyPayment:   monthly,
					MonthlyInsurance: monthlyIns,
					StartDate:        fmt.Sprintf("%02d/%d", lineStartMonth, lineStartYear),
					EndDate:          fmt.Sprintf("%02d/%d", endMonth, endYear),
					ProgressPct:      progressPct,
				})
				prop.TotalOriginalAmount += line.OriginalAmount
				prop.TotalLoanBalance += remaining
				prop.TotalAmortized += amortized
				prop.TotalMonthlyPayment += monthly
			}
			prop.NetEquity = prop.PropertyValue - prop.TotalLoanBalance
			prop.StartYear = fallbackStartYear
			prop.StartMonth = fallbackStartMonth

			// Build payment schedule reusing the existing calculator
			loanSchedule := calculator.CalculateCurrentLoanSchedule(loanLines)
			prop.PaymentSchedule = scheduleToPaymentSchedule(loanSchedule, fallbackStartYear, fallbackStartMonth)

			summary.Properties = append(summary.Properties, prop)
			summary.TotalPropertyValue += prop.PropertyValue
			summary.TotalLoanBalance += prop.TotalLoanBalance
			summary.TotalAmortized += prop.TotalAmortized
		}
	}

	// New property (nouveau bien) from NewLoanLines
	if inputs.PropertyPrice > 0 && inputs.NewLoanLines != "" {
		var newLoanLines []model.NewLoanLine
		if err := json.Unmarshal([]byte(inputs.NewLoanLines), &newLoanLines); err != nil {
			log.Printf("Failed to parse new loan lines: %v", err)
		} else if len(newLoanLines) > 0 {
			// Only show if there are actual loan lines with amounts
			hasLoans := false
			for _, line := range newLoanLines {
				if line.Amount > 0 {
					hasLoans = true
					break
				}
			}
			if hasLoans {
				prop := model.RealEstateProperty{
					Label:         "Nouveau bien",
					PropertyValue: inputs.PropertyPrice,
				}
				startYear := inputs.StartYear
				startMonth := inputs.StartMonth
				if startYear == 0 {
					startYear = atYear
					startMonth = atMonth
				}
				for _, nline := range newLoanLines {
					if nline.Amount <= 0 {
						continue
					}
					// Convert NewLoanLine to LoanLine for calculation
					line := model.LoanLine{
						Label:          nline.Label,
						OriginalAmount: nline.Amount,
						Rate:           nline.Rate,
						StartYear:      startYear,
						StartMonth:     startMonth,
						DurationYears:  nline.DurationYears,
						InsuranceRate:  nline.InsuranceRate,
						DeferralMonths: nline.DeferralMonths,
						DeferralRate:   nline.DeferralRate,
						Tiers:          nline.Tiers,
					}
					remaining, amortized, monthly, monthlyIns := calculator.ComputeLoanRemainingBalance(line, atYear, atMonth)
					var progressPct float64
					if nline.Amount > 0 {
						progressPct = amortized / nline.Amount * 100
					}

					// Compute end date: start + deferral + amortization duration
					totalMonths := nline.DurationYears*12 + nline.DeferralMonths
					endMonth := startMonth + (totalMonths % 12)
					endYear := startYear + (totalMonths / 12)
					if endMonth > 12 {
						endMonth -= 12
						endYear++
					}

					// Track slider bounds (using year*12 + (month-1) for proper decoding)
					startAbs := startYear*12 + (startMonth - 1)
					endAbs := endYear*12 + (endMonth - 1)
					if startAbs < sliderMin {
						sliderMin = startAbs
					}
					if endAbs > sliderMax {
						sliderMax = endAbs
					}

					prop.Loans = append(prop.Loans, model.RealEstateLoan{
						Label:            nline.Label,
						OriginalAmount:   nline.Amount,
						RemainingBalance: remaining,
						AmortizedCapital: amortized,
						Rate:             nline.Rate,
						MonthlyPayment:   monthly,
						MonthlyInsurance: monthlyIns,
						StartDate:        fmt.Sprintf("%02d/%d", startMonth, startYear),
						EndDate:          fmt.Sprintf("%02d/%d", endMonth, endYear),
						ProgressPct:      progressPct,
					})
					prop.TotalOriginalAmount += nline.Amount
					prop.TotalLoanBalance += remaining
					prop.TotalAmortized += amortized
					prop.TotalMonthlyPayment += monthly
				}
				prop.NetEquity = prop.PropertyValue - prop.TotalLoanBalance
				prop.StartYear = startYear
				prop.StartMonth = startMonth

				// Build payment schedule: convert NewLoanLines to LoanLines for the calculator
				var scheduleLines []model.LoanLine
				for _, nline := range newLoanLines {
					if nline.Amount <= 0 {
						continue
					}
					scheduleLines = append(scheduleLines, model.LoanLine{
						Label:          nline.Label,
						OriginalAmount: nline.Amount,
						Rate:           nline.Rate,
						DurationYears:  nline.DurationYears,
						InsuranceRate:  nline.InsuranceRate,
						DeferralMonths: nline.DeferralMonths,
						DeferralRate:   nline.DeferralRate,
						Tiers:          nline.Tiers,
					})
				}
				loanSchedule := calculator.CalculateCurrentLoanSchedule(scheduleLines)
				prop.PaymentSchedule = scheduleToPaymentSchedule(loanSchedule, startYear, startMonth)

				summary.Properties = append(summary.Properties, prop)
				summary.TotalPropertyValue += prop.PropertyValue
				summary.TotalLoanBalance += prop.TotalLoanBalance
				summary.TotalAmortized += prop.TotalAmortized
			}
		}
	}

	summary.NetEquity = summary.TotalPropertyValue - summary.TotalLoanBalance

	// Set slider values
	summary.SliderMin = sliderMin
	summary.SliderMax = sliderMax
	summary.SliderValue = atYear*12 + (atMonth - 1)
	return summary
}

func (h *PortfolioHandler) renderCashPartials(w http.ResponseWriter) {
	summary, err := h.loadCashSummary()
	if err != nil {
		log.Printf("Failed to load cash positions: %v", err)
		summary = model.ComputeCashSummary(nil)
	}

	if err := h.templates.ExecuteTemplate(w, "cash-table.html", summary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// scheduleToPaymentSchedule converts a MonthlySchedule (from calculateCurrentLoanSchedule
// or calculateTierBasedSchedule) into PaymentSchedulePoint with date labels.
func scheduleToPaymentSchedule(schedule []model.MonthlySchedule, startYear, startMonth int) []model.PaymentSchedulePoint {
	if len(schedule) == 0 {
		return nil
	}

	result := make([]model.PaymentSchedulePoint, 0, len(schedule))
	for _, ms := range schedule {
		// Compute date label from global start + month offset
		m := (startMonth - 1 + ms.Month - 1) % 12 + 1
		y := startYear + (startMonth - 1 + ms.Month - 1) / 12
		label := fmt.Sprintf("%02d/%d", m, y)

		payments := make([]model.PaymentDetail, len(ms.Payments))
		for i, p := range ms.Payments {
			payments[i] = model.PaymentDetail{
				Total:     p.Total,
				Principal: p.Principal,
				Interest:  p.Interest,
				Insurance: p.Insurance,
			}
		}

		result = append(result, model.PaymentSchedulePoint{
			Month:    ms.Month,
			Label:    label,
			Payments: payments,
		})
	}

	return result
}

func pParseFloat(s string, fallback float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}
