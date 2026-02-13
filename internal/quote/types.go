package quote

// PricePoint represents a single historical price data point.
type PricePoint struct {
	Timestamp int64   `json:"t"` // Unix milliseconds
	Price     float64 `json:"p"`
}
