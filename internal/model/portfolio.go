package model

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
