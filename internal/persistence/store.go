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
}

// Store defines the interface for persisting form inputs.
type Store interface {
	Load() (*FormInputs, error)
	Save(inputs *FormInputs) error
	Close() error
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
	}
}
