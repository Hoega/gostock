package persistence

// FormInputs contains all form fields that need to be persisted.
type FormInputs struct {
	PropertyPrice    float64
	DownPayment      float64
	InterestRate     float64
	DurationYears    int
	InsuranceRate    float64
	NotaryRate       float64
	AgencyRate       float64
	AgencyFixed      float64
	BankFees         float64
	StartYear        int
	StartMonth       int
	NetIncome1       float64
	NetIncome2       float64
	MonthlyRent      float64
	RentIncreaseRate float64
	SavingsRate      float64
	PropertyTax      float64
	CondoFees           float64
	RenovationCost      float64
	RenovationValueRate float64
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
		PropertyPrice:    250000,
		DownPayment:      0,
		InterestRate:     3.50,
		DurationYears:    20,
		InsuranceRate:    0.34,
		NotaryRate:       7.50,
		AgencyRate:       5.00,
		AgencyFixed:      0,
		BankFees:         0,
		StartYear:        2026,
		StartMonth:       1,
		NetIncome1:       0,
		NetIncome2:       0,
		MonthlyRent:         0,
		RentIncreaseRate:    2.00,
		SavingsRate:         4.00,
		RenovationValueRate: 70.00,
	}
}
