package persistence

// FormInputs contains all form fields that need to be persisted.
type FormInputs struct {
	ID                      int     `db:"id"`
	PropertyPrice           float64 `db:"property_price"`
	DownPayment             float64 `db:"down_payment"`
	InterestRate            float64 `db:"interest_rate"`
	DurationYears           int     `db:"duration_years"`
	InsuranceRate           float64 `db:"insurance_rate"`
	NotaryRate              float64 `db:"notary_rate"`
	AgencyRate              float64 `db:"agency_rate"`
	AgencyFixed             float64 `db:"agency_fixed"`
	BankFees                float64 `db:"bank_fees"`
	GuaranteeFees           float64 `db:"guarantee_fees"`
	BrokerFees              float64 `db:"broker_fees"`
	StartYear               int     `db:"start_year"`
	StartMonth              int     `db:"start_month"`
	NetIncome1              float64 `db:"net_income_1"`
	NetIncome2              float64 `db:"net_income_2"`
	MonthlyRent             float64 `db:"monthly_rent"`
	RentIncreaseRate        float64 `db:"rent_increase_rate"`
	SavingsRate             float64 `db:"savings_rate"`
	InflationRate           float64 `db:"inflation_rate"`
	PropertyTax             float64 `db:"property_tax"`
	CondoFees               float64 `db:"condo_fees"`
	MaintenanceRate         float64 `db:"maintenance_rate"`
	RenovationCost          float64 `db:"renovation_cost"`
	RenovationValueRate     float64 `db:"renovation_value_rate"`
	DownPayment1            float64 `db:"down_payment_1"`
	DownPayment2            float64 `db:"down_payment_2"`
	PaymentSplitMode        string  `db:"payment_split_mode"`
	CurrentSalePrice        float64 `db:"current_sale_price"`
	CurrentLoanBalance      float64 `db:"current_loan_balance"`
	CurrentLoanRate         float64 `db:"current_loan_rate"`
	CurrentLoanLines      string  `db:"current_loan_lines"`
	CurrentLoanStartYear  int     `db:"current_loan_start_year"`
	CurrentLoanStartMonth int     `db:"current_loan_start_month"`
	CurrentOriginalLoan   float64 `db:"current_original_loan"`
	CurrentDownPayment1     float64 `db:"current_down_payment_1"`
	CurrentRenovationCost   float64 `db:"current_renovation_cost"`
	CurrentRenovationShare2 float64 `db:"current_renovation_share_2"`
	EarlyRepaymentPenalty   float64 `db:"early_repayment_penalty"`
	SalePropertyShare1      float64 `db:"sale_property_share_1"`
	VirtualContribution2    float64 `db:"virtual_contribution_2"`
	VirtualProfitShare2     float64 `db:"virtual_profit_share_2"`
	VirtualMonthlyPayment2  float64 `db:"virtual_monthly_payment_2"`
	VirtualPaymentTiers2    string  `db:"virtual_payment_tiers_2"`
	RFRYear2_1              float64 `db:"rfr_year_2_1"`
	RFRYear1_1              float64 `db:"rfr_year_1_1"`
	RFRYear2_2              float64 `db:"rfr_year_2_2"`
	RFRYear1_2              float64 `db:"rfr_year_1_2"`
	HouseholdSize           int     `db:"household_size"`
	PropertyZone            string  `db:"property_zone"`
	NewLoanLines            string  `db:"new_loan_lines"`
	WorkLines               string  `db:"work_lines"`
	// Energy comparison fields
	Energy1Gas            float64 `db:"energy_1_gas"`
	Energy1Electricity    float64 `db:"energy_1_electricity"`
	Energy1GasKWh         float64 `db:"energy_1_gas_kwh"`
	Energy1ElectricityKWh float64 `db:"energy_1_electricity_kwh"`
	Energy1Other          float64 `db:"energy_1_other"`
	Energy1OtherLabel     string  `db:"energy_1_other_label"`
	Energy1Label          string  `db:"energy_1_label"`
	Energy2Gas            float64 `db:"energy_2_gas"`
	Energy2Electricity    float64 `db:"energy_2_electricity"`
	Energy2GasKWh         float64 `db:"energy_2_gas_kwh"`
	Energy2ElectricityKWh float64 `db:"energy_2_electricity_kwh"`
	Energy2Other          float64 `db:"energy_2_other"`
	Energy2OtherLabel     string  `db:"energy_2_other_label"`
	Energy2Label          string  `db:"energy_2_label"`
	Energy1Surface        float64 `db:"energy_1_surface"`
	Energy1DPE            float64 `db:"energy_1_dpe"`
	Energy2Surface        float64 `db:"energy_2_surface"`
	Energy2DPE            float64 `db:"energy_2_dpe"`
	Energy3Gas            float64 `db:"energy_3_gas"`
	Energy3Electricity    float64 `db:"energy_3_electricity"`
	Energy3GasKWh         float64 `db:"energy_3_gas_kwh"`
	Energy3ElectricityKWh float64 `db:"energy_3_electricity_kwh"`
	Energy3Other          float64 `db:"energy_3_other"`
	Energy3OtherLabel     string  `db:"energy_3_other_label"`
	Energy3Label          string  `db:"energy_3_label"`
	Energy3Surface        float64 `db:"energy_3_surface"`
	Energy3DPE            float64 `db:"energy_3_dpe"`
	EnergyPriceIncrease   float64 `db:"energy_price_increase"`
	// Resale projection fields
	ResaleRates     string  `db:"resale_rates"`      // JSON array
	ResaleSellCosts float64 `db:"resale_sell_costs"`
}

// Store defines the interface for persisting form inputs.
type Store interface {
	Load() (*FormInputs, error)
	Save(inputs *FormInputs) error
	LoadPositions() ([]StockPosition, error)
	SavePosition(pos *StockPosition) error
	DeletePosition(id int) error
	LoadCryptoPositions() ([]CryptoPosition, error)
	SaveCryptoPosition(pos *CryptoPosition) error
	DeleteCryptoPosition(id int) error
	// Tax sales
	LoadStockSales() ([]StockSale, error)
	LoadStockSalesByYear(year int) ([]StockSale, error)
	SaveStockSale(sale *StockSale) error
	DeleteStockSale(id int) error
	LoadCryptoSales() ([]CryptoSale, error)
	LoadCryptoSalesByYear(year int) ([]CryptoSale, error)
	SaveCryptoSale(sale *CryptoSale) error
	DeleteCryptoSale(id int) error
	GetTaxYears() ([]int, error)
	// Stock purchases for PRU calculation
	LoadStockPurchases() ([]StockPurchase, error)
	LoadStockPurchasesByISIN(isin string) ([]StockPurchase, error)
	SaveStockPurchase(purchase *StockPurchase) error
	DeleteStockPurchase(id int) error
	CalculatePRUByISIN(isin string) (float64, error)
	GetAvailableQuantityByISIN(isin string) (float64, error)
	ReduceRemainingQuantity(isin string, qty float64) error
	ResetRemainingQuantity(id int) error
	GetStockPurchaseNameByISIN(isin string) (string, string, error)
	GetEarliestPurchaseDateByISIN(isin string) (string, error)
	Close() error
}

// StockPosition represents a single stock position in a portfolio (persistence layer).
type StockPosition struct {
	ID            int     `db:"id"`
	Name          string  `db:"name"`
	ISIN          string  `db:"isin"`
	Broker        string  `db:"broker"`
	Quantity      float64 `db:"quantity"`
	PurchasePrice float64 `db:"purchase_price"`
	CurrentPrice  float64 `db:"current_price"`
	PurchaseFees  float64 `db:"purchase_fees"`
	Currency      string  `db:"currency"`
	Sector        string  `db:"sector"`
}

// CryptoPosition represents a single cryptocurrency position (persistence layer).
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

// StockSale represents a stock sale for tax reporting (2042-C).
type StockSale struct {
	ID            int     `db:"id"`
	ISIN          string  `db:"isin"`
	Name          string  `db:"name"`
	Broker        string  `db:"broker"`
	PurchaseDate  string  `db:"purchase_date"`  // YYYY-MM-DD
	PurchasePrice float64 `db:"purchase_price"` // PRU per unit
	PurchaseFees  float64 `db:"purchase_fees"`
	SaleDate      string  `db:"sale_date"` // YYYY-MM-DD
	SalePrice     float64 `db:"sale_price"`
	SaleFees      float64 `db:"sale_fees"`
	Quantity      float64 `db:"quantity"`
	Currency      string  `db:"currency"`
	TaxYear       int     `db:"tax_year"`
}

// CryptoSale represents a crypto sale for tax reporting (2086).
type CryptoSale struct {
	ID                       int     `db:"id"`
	Symbol                   string  `db:"symbol"` // BTC, ETH
	Name                     string  `db:"name"`
	Wallet                   string  `db:"wallet"`
	PurchaseDate             string  `db:"purchase_date"`
	PurchasePrice            float64 `db:"purchase_price"` // PRU per unit in EUR
	PurchaseFees             float64 `db:"purchase_fees"`
	SaleDate                 string  `db:"sale_date"`
	SalePrice                float64 `db:"sale_price"` // Price per unit at sale
	SaleFees                 float64 `db:"sale_fees"`
	Quantity                 float64 `db:"quantity"`
	PortfolioValueAtSale     float64 `db:"portfolio_value_at_sale"`     // Total portfolio value at sale time (for French method)
	PortfolioAcquisitionCost float64 `db:"portfolio_acquisition_cost"` // Total portfolio acquisition cost
	TaxYear                  int     `db:"tax_year"`
}

// StockPurchase represents a stock purchase for PRU calculation.
type StockPurchase struct {
	ID                int     `db:"id"`
	ISIN              string  `db:"isin"`
	Name              string  `db:"name"`
	Broker            string  `db:"broker"`
	Quantity          float64 `db:"quantity"`
	UnitPrice         float64 `db:"unit_price"`
	Fees              float64 `db:"fees"`
	PurchaseDate      string  `db:"purchase_date"`
	Currency          string  `db:"currency"`
	RemainingQuantity float64 `db:"remaining_quantity"`
}

// DefaultInputs returns the default form values.
func DefaultInputs() *FormInputs {
	return &FormInputs{
		ID:                  1,
		PropertyPrice:       250000,
		DownPayment:         0,
		InterestRate:        3.50,
		DurationYears:       20,
		InsuranceRate:       0.34,
		NotaryRate:          7.50,
		AgencyRate:          5.00,
		AgencyFixed:         0,
		BankFees:            0,
		StartYear:           2026,
		StartMonth:          1,
		NetIncome1:          0,
		NetIncome2:          0,
		MonthlyRent:         0,
		RentIncreaseRate:    2.00,
		SavingsRate:         4.00,
		InflationRate:       2.00,
		MaintenanceRate:     1.00,
		RenovationValueRate: 70.00,
		PaymentSplitMode:    "prorata",
		SalePropertyShare1:     50.00,
		CurrentLoanStartYear:   2020,
		CurrentLoanStartMonth:  1,
		HouseholdSize:          1,
		PropertyZone:           "B1",
		EnergyPriceIncrease:    4.0,
		ResaleRates:            "[]",
		ResaleSellCosts:        0,
	}
}
