package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store at the XDG data directory.
func NewSQLiteStore() (*SQLiteStore, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS simulation_inputs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			property_price REAL NOT NULL,
			down_payment REAL NOT NULL,
			interest_rate REAL NOT NULL,
			duration_years INTEGER NOT NULL,
			insurance_rate REAL NOT NULL,
			notary_rate REAL NOT NULL,
			agency_rate REAL NOT NULL,
			agency_fixed REAL NOT NULL,
			bank_fees REAL NOT NULL,
			start_year INTEGER NOT NULL,
			start_month INTEGER NOT NULL,
			net_income_1 REAL NOT NULL,
			net_income_2 REAL NOT NULL,
			monthly_rent REAL NOT NULL,
			rent_increase_rate REAL NOT NULL,
			savings_rate REAL NOT NULL DEFAULT 4.0,
			renovation_value_rate REAL NOT NULL DEFAULT 70.0
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Migrations: add columns if they don't exist
	_, _ = db.Exec(`ALTER TABLE simulation_inputs ADD COLUMN savings_rate REAL NOT NULL DEFAULT 4.0`)
	_, _ = db.Exec(`ALTER TABLE simulation_inputs ADD COLUMN renovation_value_rate REAL NOT NULL DEFAULT 70.0`)

	return &SQLiteStore{db: db}, nil
}

// getDBPath returns the path to the database file following XDG standards.
func getDBPath() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "gostock", "gostock.db"), nil
}

// Load retrieves the saved inputs or returns default values if none exist.
func (s *SQLiteStore) Load() (*FormInputs, error) {
	row := s.db.QueryRow(`
		SELECT property_price, down_payment, interest_rate, duration_years,
		       insurance_rate, notary_rate, agency_rate, agency_fixed,
		       bank_fees, start_year, start_month, net_income_1, net_income_2,
		       monthly_rent, rent_increase_rate, savings_rate, renovation_value_rate
		FROM simulation_inputs WHERE id = 1
	`)

	inputs := &FormInputs{}
	err := row.Scan(
		&inputs.PropertyPrice, &inputs.DownPayment, &inputs.InterestRate,
		&inputs.DurationYears, &inputs.InsuranceRate, &inputs.NotaryRate,
		&inputs.AgencyRate, &inputs.AgencyFixed, &inputs.BankFees,
		&inputs.StartYear, &inputs.StartMonth, &inputs.NetIncome1,
		&inputs.NetIncome2, &inputs.MonthlyRent, &inputs.RentIncreaseRate,
		&inputs.SavingsRate, &inputs.RenovationValueRate,
	)

	if err == sql.ErrNoRows {
		return DefaultInputs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load inputs: %w", err)
	}

	return inputs, nil
}

// Save persists the form inputs to the database.
func (s *SQLiteStore) Save(inputs *FormInputs) error {
	_, err := s.db.Exec(`
		INSERT INTO simulation_inputs (
			id, property_price, down_payment, interest_rate, duration_years,
			insurance_rate, notary_rate, agency_rate, agency_fixed,
			bank_fees, start_year, start_month, net_income_1, net_income_2,
			monthly_rent, rent_increase_rate, savings_rate, renovation_value_rate
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			property_price = excluded.property_price,
			down_payment = excluded.down_payment,
			interest_rate = excluded.interest_rate,
			duration_years = excluded.duration_years,
			insurance_rate = excluded.insurance_rate,
			notary_rate = excluded.notary_rate,
			agency_rate = excluded.agency_rate,
			agency_fixed = excluded.agency_fixed,
			bank_fees = excluded.bank_fees,
			start_year = excluded.start_year,
			start_month = excluded.start_month,
			net_income_1 = excluded.net_income_1,
			net_income_2 = excluded.net_income_2,
			monthly_rent = excluded.monthly_rent,
			rent_increase_rate = excluded.rent_increase_rate,
			savings_rate = excluded.savings_rate,
			renovation_value_rate = excluded.renovation_value_rate
	`,
		inputs.PropertyPrice, inputs.DownPayment, inputs.InterestRate,
		inputs.DurationYears, inputs.InsuranceRate, inputs.NotaryRate,
		inputs.AgencyRate, inputs.AgencyFixed, inputs.BankFees,
		inputs.StartYear, inputs.StartMonth, inputs.NetIncome1,
		inputs.NetIncome2, inputs.MonthlyRent, inputs.RentIncreaseRate,
		inputs.SavingsRate, inputs.RenovationValueRate,
	)
	if err != nil {
		return fmt.Errorf("failed to save inputs: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
