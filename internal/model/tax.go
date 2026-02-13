package model

import "math"

// StockPurchase represents a stock purchase for PRU calculation.
type StockPurchase struct {
	ID                int
	ISIN              string
	Name              string
	Broker            string
	Quantity          float64
	UnitPrice         float64
	Fees              float64
	PurchaseDate      string
	Currency          string
	RemainingQuantity float64
}

// TotalCost returns the total cost of the purchase (quantity * unit_price + fees).
func (p StockPurchase) TotalCost() float64 {
	return math.Round((p.Quantity*p.UnitPrice+p.Fees)*100) / 100
}

// StockSale represents a stock sale with computed fields for display.
type StockSale struct {
	ID            int
	ISIN          string
	Name          string
	Broker        string
	PurchaseDate  string
	PurchasePrice float64 // PRU per unit
	PurchaseFees  float64
	SaleDate      string
	SalePrice     float64
	SaleFees      float64
	Quantity      float64
	Currency      string
	TaxYear       int
}

// AcquisitionCost returns PRU * quantity + fees.
func (s StockSale) AcquisitionCost() float64 {
	return math.Round((s.PurchasePrice*s.Quantity+s.PurchaseFees)*100) / 100
}

// SaleProceeds returns sale price * quantity - fees.
func (s StockSale) SaleProceeds() float64 {
	return math.Round((s.SalePrice*s.Quantity-s.SaleFees)*100) / 100
}

// CapitalGain returns sale proceeds - acquisition cost.
func (s StockSale) CapitalGain() float64 {
	return math.Round((s.SaleProceeds()-s.AcquisitionCost())*100) / 100
}

// IsPlusValue returns true if gain >= 0.
func (s StockSale) IsPlusValue() bool {
	return s.CapitalGain() >= 0
}

// CryptoSale represents a crypto sale with computed fields for display.
type CryptoSale struct {
	ID                       int
	Symbol                   string
	Name                     string
	Wallet                   string
	PurchaseDate             string
	PurchasePrice            float64 // PRU per unit in EUR
	PurchaseFees             float64
	SaleDate                 string
	SalePrice                float64 // Price per unit at sale
	SaleFees                 float64
	Quantity                 float64
	PortfolioValueAtSale     float64 // Total portfolio value at sale (for French global method)
	PortfolioAcquisitionCost float64 // Total portfolio acquisition cost
	TaxYear                  int
}

// SaleProceeds returns the gross proceeds from the sale.
func (c CryptoSale) SaleProceeds() float64 {
	return math.Round((c.SalePrice*c.Quantity-c.SaleFees)*100) / 100
}

// AcquisitionCost returns the proportional acquisition cost using French global method.
// Formula: (Portfolio acquisition cost) * (Sale proceeds / Portfolio value at sale)
func (c CryptoSale) AcquisitionCost() float64 {
	if c.PortfolioValueAtSale == 0 {
		// Fallback to simple method if portfolio value not provided
		return math.Round((c.PurchasePrice*c.Quantity+c.PurchaseFees)*100) / 100
	}
	proportion := c.SaleProceeds() / c.PortfolioValueAtSale
	return math.Round((c.PortfolioAcquisitionCost*proportion)*100) / 100
}

// CapitalGain returns the taxable gain using French method.
func (c CryptoSale) CapitalGain() float64 {
	return math.Round((c.SaleProceeds()-c.AcquisitionCost())*100) / 100
}

// IsPlusValue returns true if gain >= 0.
func (c CryptoSale) IsPlusValue() bool {
	return c.CapitalGain() >= 0
}

// TaxYearSummary aggregates capital gains/losses for a tax year.
type TaxYearSummary struct {
	Year int

	// Stocks (2042-C)
	StockSales        []StockSale
	StockTotalPV      float64 // Total plus-values
	StockTotalMV      float64 // Total moins-values
	StockNetResult    float64 // PV - MV
	StockTaxableBase  float64 // Base imposable after MV carryforward

	// Crypto (2086)
	CryptoSales          []CryptoSale
	CryptoTotalCessions  float64 // Total cessions (3AN)
	CryptoTotalPV        float64 // Total plus-values
	CryptoTotalMV        float64 // Total moins-values
	CryptoNetResult      float64 // PV - MV
	CryptoTaxableBase    float64 // Case 3BN

	// Combined
	TotalTaxableBase float64 // Total PFU base
	PFUAmount        float64 // 30% flat tax amount
}

// StockSaleSummary holds summary data for display.
type StockSaleSummary struct {
	Sales         []StockSale
	Purchases     []StockPurchase
	TotalPV       float64
	TotalMV       float64
	NetResult     float64
	TaxableBase   float64
	PFUAmount     float64
}

// CryptoSaleSummary holds summary data for crypto display.
type CryptoSaleSummary struct {
	Sales            []CryptoSale
	TotalCessions    float64 // Case 3AN
	TotalPV          float64
	TotalMV          float64
	NetResult        float64
	TaxableBase      float64 // Case 3BN
	PFUAmount        float64
}

// TaxPageData holds all data for the tax page.
type TaxPageData struct {
	ActiveTab     string // "stocks" or "crypto"
	SelectedYear  int
	AvailableYears []int
	StockSummary  StockSaleSummary
	CryptoSummary CryptoSaleSummary
}
