package model

import "sort"

// StockPosition represents a single stock position in a portfolio.
type StockPosition struct {
	ID            int     `db:"id"`
	Name          string  `db:"name"`
	ISIN          string  `db:"isin"`
	Broker        string  `db:"broker"` // "boursobank" or "degiro"
	Quantity      float64 `db:"quantity"`
	PurchasePrice float64 `db:"purchase_price"` // PRU
	CurrentPrice  float64 `db:"current_price"`
	PurchaseFees  float64 `db:"purchase_fees"`
	Currency      string  `db:"currency"` // Default: EUR
	Sector        string  `db:"sector"`
	// Computed EUR-converted values (populated by ComputePortfolioSummary)
	ValueEUR    float64
	CostEUR     float64
	GainLossEUR float64
}

// TotalValue returns the current market value of the position.
func (p StockPosition) TotalValue() float64 {
	return p.Quantity * p.CurrentPrice
}

// TotalCost returns the total purchase cost including fees.
func (p StockPosition) TotalCost() float64 {
	return p.Quantity*p.PurchasePrice + p.PurchaseFees
}

// GainLoss returns the absolute gain/loss in euros.
func (p StockPosition) GainLoss() float64 {
	return p.TotalValue() - p.TotalCost()
}

// GainLossPct returns the gain/loss percentage.
func (p StockPosition) GainLossPct() float64 {
	cost := p.TotalCost()
	if cost == 0 {
		return 0
	}
	return (p.GainLoss() / cost) * 100
}

// BrokerSummary holds aggregated data for one broker.
type BrokerSummary struct {
	Broker     string
	TotalValue float64
	TotalCost  float64
	GainLoss   float64
	GainPct    float64
	Count      int
}

// DisplayName returns the human-readable broker name.
func (b BrokerSummary) DisplayName() string {
	return brokerDisplayName(b.Broker)
}

// PortfolioSummary holds computed totals for the entire portfolio.
type PortfolioSummary struct {
	Positions      []StockPosition
	Brokers        []BrokerSummary
	TotalValue     float64
	TotalCost      float64
	TotalGainLoss  float64
	TotalGainPct   float64
	ExchangeRates  map[string]float64 // currency → multiplier to EUR (e.g. "USD": 0.926)
	// Chart data
	BrokerLabels   []string
	BrokerValues   []float64
	StockLabels    []string
	StockValues    []float64
	GainLabels     []string
	GainValues     []float64
	GainColors     []string
}

// ComputePortfolioSummary calculates the summary from a list of positions.
// rates maps currency codes to their EUR multiplier (e.g. {"EUR": 1.0, "USD": 0.926}).
// If rates is nil, all positions are treated as EUR (multiplier = 1).
func ComputePortfolioSummary(positions []StockPosition, rates map[string]float64) PortfolioSummary {
	s := PortfolioSummary{Positions: positions, ExchangeRates: rates}

	brokerMap := make(map[string]*BrokerSummary)

	for i, p := range positions {
		val := p.TotalValue()
		cost := p.TotalCost()
		gl := p.GainLoss()

		// Convert to EUR
		rate := 1.0
		if rates != nil {
			if r, ok := rates[p.Currency]; ok {
				rate = r
			}
		}
		valEUR := val * rate
		costEUR := cost * rate
		glEUR := gl * rate

		s.Positions[i].ValueEUR = valEUR
		s.Positions[i].CostEUR = costEUR
		s.Positions[i].GainLossEUR = glEUR

		s.TotalValue += valEUR
		s.TotalCost += costEUR
		s.TotalGainLoss += glEUR

		bs, ok := brokerMap[p.Broker]
		if !ok {
			bs = &BrokerSummary{Broker: p.Broker}
			brokerMap[p.Broker] = bs
		}
		bs.TotalValue += valEUR
		bs.TotalCost += costEUR
		bs.GainLoss += glEUR
		bs.Count++

		// Stock allocation chart data (in EUR)
		s.StockLabels = append(s.StockLabels, p.Name)
		s.StockValues = append(s.StockValues, valEUR)

		// Gain/loss bar chart data (in EUR)
		s.GainLabels = append(s.GainLabels, p.Name)
		s.GainValues = append(s.GainValues, glEUR)
		if glEUR >= 0 {
			s.GainColors = append(s.GainColors, "rgba(34, 197, 94, 0.7)")
		} else {
			s.GainColors = append(s.GainColors, "rgba(239, 68, 68, 0.7)")
		}
	}

	if s.TotalCost > 0 {
		s.TotalGainPct = (s.TotalGainLoss / s.TotalCost) * 100
	}

	// Build broker summaries and chart data
	for _, name := range []string{"boursobank", "degiro"} {
		if bs, ok := brokerMap[name]; ok {
			if bs.TotalCost > 0 {
				bs.GainPct = (bs.GainLoss / bs.TotalCost) * 100
			}
			s.Brokers = append(s.Brokers, *bs)
			s.BrokerLabels = append(s.BrokerLabels, brokerDisplayName(name))
			s.BrokerValues = append(s.BrokerValues, bs.TotalValue)
		}
	}

	return s
}

func brokerDisplayName(b string) string {
	switch b {
	case "boursobank":
		return "Boursobank"
	case "degiro":
		return "Degiro"
	default:
		return b
	}
}

// CashPosition represents a cash position in a bank account.
type CashPosition struct {
	ID              int
	BankName        string
	Amount          float64
	AccountType     string
	InterestRate    float64
	AnnualReturn    float64 // Computed: Amount * InterestRate / 100
}

// AccountTypeDisplay returns a human-readable account type label.
func (p CashPosition) AccountTypeDisplay() string {
	return accountTypeDisplayName(p.AccountType)
}

func accountTypeDisplayName(t string) string {
	switch t {
	case "courant":
		return "Compte courant"
	case "livret_a":
		return "Livret A"
	case "ldds":
		return "LDDS"
	case "lep":
		return "LEP"
	case "pel":
		return "PEL"
	case "cel":
		return "CEL"
	case "assurance_vie":
		return "Assurance vie (fonds euros)"
	case "compte_terme":
		return "Compte à terme"
	default:
		return t
	}
}

// BankSummary holds aggregated data for one bank.
type BankSummary struct {
	BankName     string
	TotalAmount  float64
	AnnualReturn float64
	Count        int
}

// CashSummary holds computed totals for the cash portfolio.
type CashSummary struct {
	Positions    []CashPosition
	Banks        []BankSummary
	TotalAmount  float64
	AnnualReturn float64
	// Chart data
	BankLabels   []string
	BankValues   []float64
	TypeLabels   []string
	TypeValues   []float64
}

// ComputeCashSummary calculates the summary from a list of cash positions.
func ComputeCashSummary(positions []CashPosition) CashSummary {
	s := CashSummary{Positions: positions}

	bankMap := make(map[string]*BankSummary)
	typeAlloc := make(map[string]float64)
	var typeOrder []string

	for i, p := range positions {
		annualReturn := p.Amount * p.InterestRate / 100
		s.Positions[i].AnnualReturn = annualReturn

		s.TotalAmount += p.Amount
		s.AnnualReturn += annualReturn

		bs, ok := bankMap[p.BankName]
		if !ok {
			bs = &BankSummary{BankName: p.BankName}
			bankMap[p.BankName] = bs
		}
		bs.TotalAmount += p.Amount
		bs.AnnualReturn += annualReturn
		bs.Count++

		// Type allocation chart data
		typeLabel := accountTypeDisplayName(p.AccountType)
		if _, seen := typeAlloc[typeLabel]; !seen {
			typeOrder = append(typeOrder, typeLabel)
		}
		typeAlloc[typeLabel] += p.Amount
	}

	// Build type allocation chart slices
	for _, t := range typeOrder {
		s.TypeLabels = append(s.TypeLabels, t)
		s.TypeValues = append(s.TypeValues, typeAlloc[t])
	}

	// Build bank summaries and chart data (sorted by name)
	var bankNames []string
	for name := range bankMap {
		bankNames = append(bankNames, name)
	}
	sort.Strings(bankNames)
	for _, name := range bankNames {
		bs := bankMap[name]
		s.Banks = append(s.Banks, *bs)
		s.BankLabels = append(s.BankLabels, name)
		s.BankValues = append(s.BankValues, bs.TotalAmount)
	}

	return s
}

// CryptoPosition represents a single cryptocurrency position.
type CryptoPosition struct {
	ID            int     `db:"id"`
	Symbol        string  `db:"symbol"`         // BTC, ETH
	CoingeckoID   string  `db:"coingecko_id"`   // bitcoin, ethereum
	Name          string  `db:"name"`
	Wallet        string  `db:"wallet"`         // ledger, binance, kraken
	Quantity      float64 `db:"quantity"`
	PurchasePrice float64 `db:"purchase_price"` // PRU in EUR
	CurrentPrice  float64 `db:"current_price"`
	PurchaseFees  float64 `db:"purchase_fees"`
}

// HasPRU returns true if the position has a known cost basis (PRU > 0).
func (p CryptoPosition) HasPRU() bool {
	return p.PurchasePrice > 0
}

// TotalValue returns the current market value of the position in EUR.
func (p CryptoPosition) TotalValue() float64 {
	return p.Quantity * p.CurrentPrice
}

// TotalCost returns the total purchase cost including fees in EUR.
func (p CryptoPosition) TotalCost() float64 {
	return p.Quantity*p.PurchasePrice + p.PurchaseFees
}

// GainLoss returns the absolute gain/loss in EUR.
func (p CryptoPosition) GainLoss() float64 {
	return p.TotalValue() - p.TotalCost()
}

// GainLossPct returns the gain/loss percentage.
func (p CryptoPosition) GainLossPct() float64 {
	cost := p.TotalCost()
	if cost == 0 {
		return 0
	}
	return (p.GainLoss() / cost) * 100
}

// WalletSummary holds aggregated data for one wallet.
type WalletSummary struct {
	Wallet     string
	TotalValue float64
	TotalCost  float64
	GainLoss   float64
	GainPct    float64
	Count      int
}

// DisplayName returns the human-readable wallet name.
func (w WalletSummary) DisplayName() string {
	return walletDisplayName(w.Wallet)
}

func walletDisplayName(w string) string {
	switch w {
	case "binance":
		return "Binance"
	case "coinbase":
		return "Coinbase"
	case "cryptocom":
		return "Crypto.com"
	case "swissborg":
		return "SwissBorg"
	default:
		return w
	}
}

// CryptoSummary holds computed totals for the crypto portfolio.
type CryptoSummary struct {
	Positions     []CryptoPosition
	Wallets       []WalletSummary
	TotalValue    float64
	TotalCost     float64
	TotalGainLoss float64
	TotalGainPct  float64
	// Chart data
	WalletLabels  []string
	WalletValues  []float64
	CryptoLabels  []string
	CryptoValues  []float64
	GainLabels    []string
	GainValues    []float64
	GainColors    []string
}

// PaymentDetail holds the breakdown of a single loan payment for one month.
type PaymentDetail struct {
	Total     float64 // Total pour ce prêt (capital + intérêts + assurance)
	Principal float64 // Capital remboursé
	Interest  float64 // Intérêts (ou intérêts intercalaires en période de différé)
	Insurance float64 // Assurance
}

// PaymentSchedulePoint holds payment data for one month across all loans of a property.
type PaymentSchedulePoint struct {
	Month    int             // mois absolu (1-based depuis le début du premier prêt)
	Label    string          // "01/2021"
	Payments []PaymentDetail // détail par loan (même ordre que Loans)
}

// RealEstateLoan represents a single real estate loan with amortization progress.
type RealEstateLoan struct {
	Label            string
	OriginalAmount   float64
	RemainingBalance float64
	AmortizedCapital float64
	Rate             float64
	MonthlyPayment   float64
	MonthlyInsurance float64
	StartDate        string
	EndDate          string
	ProgressPct      float64 // % du capital amorti
}

// RealEstateProperty represents a single real estate property.
type RealEstateProperty struct {
	Label               string
	PropertyValue       float64
	Loans               []RealEstateLoan
	TotalOriginalAmount float64
	TotalLoanBalance    float64
	TotalAmortized      float64
	TotalMonthlyPayment float64
	NetEquity           float64
	PaymentSchedule     []PaymentSchedulePoint
	StartYear           int
	StartMonth          int
}

// RealEstateSummary holds the summary of real estate patrimony.
type RealEstateSummary struct {
	Properties         []RealEstateProperty
	TotalPropertyValue float64
	TotalLoanBalance   float64
	TotalAmortized     float64
	NetEquity          float64
	// Slider data (absolute months = year*12 + month)
	AtYear   int // currently displayed year
	AtMonth  int // currently displayed month
	SliderMin   int // earliest loan start (absolute month)
	SliderMax   int // latest loan end (absolute month)
	SliderValue int // current position (absolute month)
}

// GlobalPosition represents a single position from any asset class.
type GlobalPosition struct {
	Name    string
	Type    string  // "stock", "crypto", "cash"
	Value   float64
	Percent float64 // % of grand total
}

// GlobalSummary holds aggregated data across all asset classes.
type GlobalSummary struct {
	StockTotal      float64
	CryptoTotal     float64
	CashTotal       float64
	RealEstateTotal         float64
	RealEstateAmortized     float64
	RealEstatePropertyValue float64
	GrandTotal              float64
	// Chart data
	ClassLabels []string  // ["Actions", "Crypto", "Cash", "Immobilier"]
	ClassValues []float64
	// Top positions (all types merged)
	TopPositions []GlobalPosition
}

// ComputeGlobalSummary aggregates stock, crypto, cash and real estate summaries into a global overview.
func ComputeGlobalSummary(stocks PortfolioSummary, crypto CryptoSummary, cash CashSummary, realEstate RealEstateSummary) GlobalSummary {
	g := GlobalSummary{
		StockTotal:      stocks.TotalValue,
		CryptoTotal:     crypto.TotalValue,
		CashTotal:       cash.TotalAmount,
		RealEstateTotal:         realEstate.NetEquity,
		RealEstateAmortized:     realEstate.TotalAmortized,
		RealEstatePropertyValue: realEstate.TotalPropertyValue,
	}
	g.GrandTotal = g.StockTotal + g.CryptoTotal + g.CashTotal + g.RealEstateTotal

	// Class allocation chart
	initialPlusValue := g.RealEstateTotal - g.RealEstateAmortized
	g.ClassLabels = []string{"Actions", "Crypto", "Cash", "Immo. Capital amorti", "Immo. Plus-value"}
	g.ClassValues = []float64{g.StockTotal, g.CryptoTotal, g.CashTotal, g.RealEstateAmortized, initialPlusValue}

	// Merge all positions into a single list
	var all []GlobalPosition
	for _, p := range stocks.Positions {
		all = append(all, GlobalPosition{
			Name:  p.Name,
			Type:  "stock",
			Value: p.ValueEUR,
		})
	}
	// Aggregate crypto by symbol (may appear in multiple wallets)
	cryptoBySymbol := make(map[string]float64)
	var cryptoOrder []string
	for _, p := range crypto.Positions {
		if _, seen := cryptoBySymbol[p.Symbol]; !seen {
			cryptoOrder = append(cryptoOrder, p.Symbol)
		}
		cryptoBySymbol[p.Symbol] += p.TotalValue()
	}
	for _, sym := range cryptoOrder {
		all = append(all, GlobalPosition{
			Name:  sym,
			Type:  "crypto",
			Value: cryptoBySymbol[sym],
		})
	}
	for _, p := range cash.Positions {
		label := p.BankName + " - " + accountTypeDisplayName(p.AccountType)
		all = append(all, GlobalPosition{
			Name:  label,
			Type:  "cash",
			Value: p.Amount,
		})
	}
	for _, p := range realEstate.Properties {
		all = append(all, GlobalPosition{
			Name:  p.Label,
			Type:  "immobilier",
			Value: p.NetEquity,
		})
	}

	// Sort by value descending
	sort.Slice(all, func(i, j int) bool {
		return all[i].Value > all[j].Value
	})

	// Compute percent and keep top 10
	limit := min(10, len(all))
	for i := 0; i < limit; i++ {
		if g.GrandTotal > 0 {
			all[i].Percent = all[i].Value / g.GrandTotal * 100
		}
		g.TopPositions = append(g.TopPositions, all[i])
	}

	return g
}

// ComputeCryptoSummary calculates the summary from a list of crypto positions.
func ComputeCryptoSummary(positions []CryptoPosition) CryptoSummary {
	s := CryptoSummary{Positions: positions}

	walletMap := make(map[string]*WalletSummary)
	cryptoAlloc := make(map[string]float64)  // symbol → total value (aggregated across wallets)
	var cryptoOrder []string                  // preserve first-seen order

	for _, p := range positions {
		val := p.TotalValue()

		s.TotalValue += val

		ws, ok := walletMap[p.Wallet]
		if !ok {
			ws = &WalletSummary{Wallet: p.Wallet}
			walletMap[p.Wallet] = ws
		}
		ws.TotalValue += val
		ws.Count++

		// Only include cost/gain for positions with a known PRU
		if p.HasPRU() {
			cost := p.TotalCost()
			gl := p.GainLoss()

			s.TotalCost += cost
			s.TotalGainLoss += gl

			ws.TotalCost += cost
			ws.GainLoss += gl

			// Gain/loss bar chart data (only for positions with PRU)
			s.GainLabels = append(s.GainLabels, p.Symbol)
			s.GainValues = append(s.GainValues, gl)
			if gl >= 0 {
				s.GainColors = append(s.GainColors, "rgba(34, 197, 94, 0.7)")
			} else {
				s.GainColors = append(s.GainColors, "rgba(239, 68, 68, 0.7)")
			}
		}

		// Crypto allocation chart data — aggregate by symbol across wallets
		if _, seen := cryptoAlloc[p.Symbol]; !seen {
			cryptoOrder = append(cryptoOrder, p.Symbol)
		}
		cryptoAlloc[p.Symbol] += val
	}

	// Build crypto allocation chart slices in first-seen order
	for _, sym := range cryptoOrder {
		s.CryptoLabels = append(s.CryptoLabels, sym)
		s.CryptoValues = append(s.CryptoValues, cryptoAlloc[sym])
	}

	if s.TotalCost > 0 {
		s.TotalGainPct = (s.TotalGainLoss / s.TotalCost) * 100
	}

	// Build wallet summaries and chart data (ordered)
	walletOrder := []string{"binance", "coinbase", "cryptocom", "swissborg"}
	seenWallets := make(map[string]bool)
	for _, name := range walletOrder {
		if ws, ok := walletMap[name]; ok {
			if ws.TotalCost > 0 {
				ws.GainPct = (ws.GainLoss / ws.TotalCost) * 100
			}
			s.Wallets = append(s.Wallets, *ws)
			s.WalletLabels = append(s.WalletLabels, walletDisplayName(name))
			s.WalletValues = append(s.WalletValues, ws.TotalValue)
			seenWallets[name] = true
		}
	}
	// Add any other wallets not in the predefined order
	for name, ws := range walletMap {
		if !seenWallets[name] {
			if ws.TotalCost > 0 {
				ws.GainPct = (ws.GainLoss / ws.TotalCost) * 100
			}
			s.Wallets = append(s.Wallets, *ws)
			s.WalletLabels = append(s.WalletLabels, walletDisplayName(name))
			s.WalletValues = append(s.WalletValues, ws.TotalValue)
		}
	}

	return s
}
