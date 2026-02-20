package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
	"github.com/Hoega/gostock/internal/quote"
)

type TaxHandler struct {
	templates *template.Template
	store     persistence.Store
}

func NewTaxHandler(templates *template.Template, store persistence.Store) *TaxHandler {
	return &TaxHandler{templates: templates, store: store}
}

// ShowTax renders the tax page with stock and crypto sales.
func (h *TaxHandler) ShowTax(w http.ResponseWriter, r *http.Request) {
	activeTab := r.URL.Query().Get("tab")
	if activeTab != "crypto" {
		activeTab = "stocks"
	}

	// Get selected year from query, default to current year
	selectedYear := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			selectedYear = y
		}
	}

	// Get available years
	availableYears, err := h.store.GetTaxYears()
	if err != nil {
		log.Printf("Failed to get tax years: %v", err)
		availableYears = []int{selectedYear}
	}
	// Ensure current year is in the list
	if len(availableYears) == 0 || !containsYear(availableYears, selectedYear) {
		availableYears = append([]int{selectedYear}, availableYears...)
	}

	data := model.TaxPageData{
		ActiveTab:      activeTab,
		SelectedYear:   selectedYear,
		AvailableYears: availableYears,
	}

	if activeTab == "stocks" {
		data.StockSummary = h.loadStockSummary(selectedYear)
	} else {
		data.CryptoSummary = h.loadCryptoSummary(selectedYear)
	}

	// If HTMX request, render only the container partial
	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "tax-container.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.ExecuteTemplate(w, "tax.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AddStockSale handles POST /tax/stocks/sales.
func (h *TaxHandler) AddStockSale(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	sale := h.parseStockSaleForm(r)

	// Reduce remaining quantities if ISIN is provided
	if sale.ISIN != "" {
		if err := h.store.ReduceRemainingQuantity(sale.ISIN, sale.Quantity); err != nil {
			log.Printf("Failed to reduce remaining quantity for ISIN %s: %v", sale.ISIN, err)
			// Continue anyway - the sale should still be recorded
		}
	}

	if err := h.store.SaveStockSale(sale); err != nil {
		log.Printf("Failed to save stock sale: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w, sale.TaxYear)
}

// UpdateStockSale handles PUT /tax/stocks/sales/{id}.
func (h *TaxHandler) UpdateStockSale(w http.ResponseWriter, r *http.Request) {
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

	sale := h.parseStockSaleForm(r)
	sale.ID = id
	if err := h.store.SaveStockSale(sale); err != nil {
		log.Printf("Failed to update stock sale: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w, sale.TaxYear)
}

// DeleteStockSale handles DELETE /tax/stocks/sales/{id}.
func (h *TaxHandler) DeleteStockSale(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	if err := h.store.DeleteStockSale(id); err != nil {
		log.Printf("Failed to delete stock sale: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w, year)
}

// AddCryptoSale handles POST /tax/crypto/sales.
func (h *TaxHandler) AddCryptoSale(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	sale := h.parseCryptoSaleForm(r)
	if err := h.store.SaveCryptoSale(sale); err != nil {
		log.Printf("Failed to save crypto sale: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w, sale.TaxYear)
}

// UpdateCryptoSale handles PUT /tax/crypto/sales/{id}.
func (h *TaxHandler) UpdateCryptoSale(w http.ResponseWriter, r *http.Request) {
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

	sale := h.parseCryptoSaleForm(r)
	sale.ID = id
	if err := h.store.SaveCryptoSale(sale); err != nil {
		log.Printf("Failed to update crypto sale: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w, sale.TaxYear)
}

// DeleteCryptoSale handles DELETE /tax/crypto/sales/{id}.
func (h *TaxHandler) DeleteCryptoSale(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	if err := h.store.DeleteCryptoSale(id); err != nil {
		log.Printf("Failed to delete crypto sale: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderCryptoPartials(w, year)
}

// AddStockPurchase handles POST /tax/purchases.
func (h *TaxHandler) AddStockPurchase(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	purchase := h.parseStockPurchaseForm(r)
	if err := h.store.SaveStockPurchase(purchase); err != nil {
		log.Printf("Failed to save stock purchase: %v", err)
		http.Error(w, "Erreur lors de la sauvegarde", http.StatusInternalServerError)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	h.renderStockPartials(w, year)
}

// UpdateStockPurchase handles PUT /tax/purchases/{id}.
func (h *TaxHandler) UpdateStockPurchase(w http.ResponseWriter, r *http.Request) {
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

	purchase := h.parseStockPurchaseForm(r)
	purchase.ID = id
	// Keep remaining quantity from form (might have been edited)
	if remaining := r.FormValue("remaining_quantity"); remaining != "" {
		purchase.RemainingQuantity = taxParseFloat(remaining, purchase.Quantity)
	}
	if err := h.store.SaveStockPurchase(purchase); err != nil {
		log.Printf("Failed to update stock purchase: %v", err)
		http.Error(w, "Erreur lors de la mise à jour", http.StatusInternalServerError)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	h.renderStockPartials(w, year)
}

// DeleteStockPurchase handles DELETE /tax/purchases/{id}.
func (h *TaxHandler) DeleteStockPurchase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	if err := h.store.DeleteStockPurchase(id); err != nil {
		log.Printf("Failed to delete stock purchase: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w, year)
}

// ResetStockPurchase handles POST /tax/purchases/{id}/reset to restore remaining quantity.
func (h *TaxHandler) ResetStockPurchase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Get year from query for rendering
	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	if err := h.store.ResetRemainingQuantity(id); err != nil {
		log.Printf("Failed to reset stock purchase: %v", err)
		http.Error(w, "Erreur lors de la réinitialisation", http.StatusInternalServerError)
		return
	}

	h.renderStockPartials(w, year)
}

// GetPRUForISIN handles GET /tax/purchases/pru?isin=XXX.
func (h *TaxHandler) GetPRUForISIN(w http.ResponseWriter, r *http.Request) {
	isin := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("isin")))
	if isin == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pru":      0,
			"quantity": 0,
		})
		return
	}

	pru, err := h.store.CalculatePRUByISIN(isin)
	if err != nil {
		log.Printf("Failed to calculate PRU for ISIN %s: %v", isin, err)
		pru = 0
	}

	availableQty, err := h.store.GetAvailableQuantityByISIN(isin)
	if err != nil {
		log.Printf("Failed to get available quantity for ISIN %s: %v", isin, err)
		availableQty = 0
	}

	name, broker, err := h.store.GetStockPurchaseNameByISIN(isin)
	if err != nil {
		// No purchase found at all, try Yahoo lookup
		result, lookupErr := quote.LookupISIN(isin)
		if lookupErr == nil && result != nil {
			name = result.Name
		}
	}

	purchaseDate, _ := h.store.GetEarliestPurchaseDateByISIN(isin)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pru":           pru,
		"quantity":      availableQty,
		"name":          name,
		"broker":        broker,
		"purchase_date": purchaseDate,
	})
}

func (h *TaxHandler) parseStockPurchaseForm(r *http.Request) *persistence.StockPurchase {
	currency := r.FormValue("currency")
	if currency == "" {
		currency = "EUR"
	}

	return &persistence.StockPurchase{
		ISIN:         strings.ToUpper(strings.TrimSpace(r.FormValue("isin"))),
		Name:         r.FormValue("name"),
		Broker:       r.FormValue("broker"),
		Quantity:     taxParseFloat(r.FormValue("quantity"), 0),
		UnitPrice:    taxParseFloat(r.FormValue("unit_price"), 0),
		Fees:         taxParseFloat(r.FormValue("fees"), 0),
		PurchaseDate: r.FormValue("purchase_date"),
		Currency:     currency,
	}
}

func (h *TaxHandler) parseStockSaleForm(r *http.Request) *persistence.StockSale {
	currency := r.FormValue("currency")
	if currency == "" {
		currency = "EUR"
	}

	// Extract year from sale_date
	taxYear := time.Now().Year()
	if saleDate := r.FormValue("sale_date"); saleDate != "" {
		if t, err := time.Parse("2006-01-02", saleDate); err == nil {
			taxYear = t.Year()
		}
	}

	return &persistence.StockSale{
		ISIN:          r.FormValue("isin"),
		Name:          r.FormValue("name"),
		Broker:        r.FormValue("broker"),
		PurchaseDate:  r.FormValue("purchase_date"),
		PurchasePrice: taxParseFloat(r.FormValue("purchase_price"), 0),
		PurchaseFees:  taxParseFloat(r.FormValue("purchase_fees"), 0),
		SaleDate:      r.FormValue("sale_date"),
		SalePrice:     taxParseFloat(r.FormValue("sale_price"), 0),
		SaleFees:      taxParseFloat(r.FormValue("sale_fees"), 0),
		Quantity:      taxParseFloat(r.FormValue("quantity"), 0),
		Currency:      currency,
		TaxYear:       taxYear,
	}
}

func (h *TaxHandler) parseCryptoSaleForm(r *http.Request) *persistence.CryptoSale {
	// Extract year from sale_date
	taxYear := time.Now().Year()
	if saleDate := r.FormValue("sale_date"); saleDate != "" {
		if t, err := time.Parse("2006-01-02", saleDate); err == nil {
			taxYear = t.Year()
		}
	}

	return &persistence.CryptoSale{
		Symbol:                   r.FormValue("symbol"),
		Name:                     r.FormValue("name"),
		Wallet:                   r.FormValue("wallet"),
		PurchaseDate:             r.FormValue("purchase_date"),
		PurchasePrice:            taxParseFloat(r.FormValue("purchase_price"), 0),
		PurchaseFees:             taxParseFloat(r.FormValue("purchase_fees"), 0),
		SaleDate:                 r.FormValue("sale_date"),
		SalePrice:                taxParseFloat(r.FormValue("sale_price"), 0),
		SaleFees:                 taxParseFloat(r.FormValue("sale_fees"), 0),
		Quantity:                 taxParseFloat(r.FormValue("quantity"), 0),
		PortfolioValueAtSale:     taxParseFloat(r.FormValue("portfolio_value"), 0),
		PortfolioAcquisitionCost: taxParseFloat(r.FormValue("portfolio_cost"), 0),
		TaxYear:                  taxYear,
	}
}

func (h *TaxHandler) loadStockSummary(year int) model.StockSaleSummary {
	dbSales, err := h.store.LoadStockSalesByYear(year)
	if err != nil {
		log.Printf("Failed to load stock sales: %v", err)
		return calculator.ComputeStockSaleSummary(nil, nil)
	}

	sales := make([]model.StockSale, len(dbSales))
	for i, s := range dbSales {
		sales[i] = model.StockSale{
			ID:            s.ID,
			ISIN:          s.ISIN,
			Name:          s.Name,
			Broker:        s.Broker,
			PurchaseDate:  s.PurchaseDate,
			PurchasePrice: s.PurchasePrice,
			PurchaseFees:  s.PurchaseFees,
			SaleDate:      s.SaleDate,
			SalePrice:     s.SalePrice,
			SaleFees:      s.SaleFees,
			Quantity:      s.Quantity,
			Currency:      s.Currency,
			TaxYear:       s.TaxYear,
		}
	}

	// Load purchases
	dbPurchases, err := h.store.LoadStockPurchases()
	if err != nil {
		log.Printf("Failed to load stock purchases: %v", err)
		return calculator.ComputeStockSaleSummary(sales, nil)
	}

	purchases := make([]model.StockPurchase, len(dbPurchases))
	for i, p := range dbPurchases {
		purchases[i] = model.StockPurchase{
			ID:                p.ID,
			ISIN:              p.ISIN,
			Name:              p.Name,
			Broker:            p.Broker,
			Quantity:          p.Quantity,
			UnitPrice:         p.UnitPrice,
			Fees:              p.Fees,
			PurchaseDate:      p.PurchaseDate,
			Currency:          p.Currency,
			RemainingQuantity: p.RemainingQuantity,
		}
	}

	return calculator.ComputeStockSaleSummary(sales, purchases)
}

func (h *TaxHandler) loadCryptoSummary(year int) model.CryptoSaleSummary {
	dbSales, err := h.store.LoadCryptoSalesByYear(year)
	if err != nil {
		log.Printf("Failed to load crypto sales: %v", err)
		return calculator.ComputeCryptoSaleSummary(nil)
	}

	sales := make([]model.CryptoSale, len(dbSales))
	for i, s := range dbSales {
		sales[i] = model.CryptoSale{
			ID:                       s.ID,
			Symbol:                   s.Symbol,
			Name:                     s.Name,
			Wallet:                   s.Wallet,
			PurchaseDate:             s.PurchaseDate,
			PurchasePrice:            s.PurchasePrice,
			PurchaseFees:             s.PurchaseFees,
			SaleDate:                 s.SaleDate,
			SalePrice:                s.SalePrice,
			SaleFees:                 s.SaleFees,
			Quantity:                 s.Quantity,
			PortfolioValueAtSale:     s.PortfolioValueAtSale,
			PortfolioAcquisitionCost: s.PortfolioAcquisitionCost,
			TaxYear:                  s.TaxYear,
		}
	}

	return calculator.ComputeCryptoSaleSummary(sales)
}

func (h *TaxHandler) renderStockPartials(w http.ResponseWriter, year int) {
	summary := h.loadStockSummary(year)

	data := struct {
		StockSummary model.StockSaleSummary
		SelectedYear int
	}{
		StockSummary: summary,
		SelectedYear: year,
	}

	if err := h.templates.ExecuteTemplate(w, "tax-stock-table.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "tax-summary.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TaxHandler) renderCryptoPartials(w http.ResponseWriter, year int) {
	summary := h.loadCryptoSummary(year)

	data := struct {
		CryptoSummary model.CryptoSaleSummary
		SelectedYear  int
	}{
		CryptoSummary: summary,
		SelectedYear:  year,
	}

	if err := h.templates.ExecuteTemplate(w, "tax-crypto-table.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "tax-summary.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func taxParseFloat(s string, fallback float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func containsYear(years []int, year int) bool {
	for _, y := range years {
		if y == year {
			return true
		}
	}
	return false
}
