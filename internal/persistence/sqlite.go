package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sqlx.DB
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

	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS simulation_inputs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			property_price REAL NOT NULL DEFAULT 250000,
			down_payment REAL NOT NULL DEFAULT 0,
			interest_rate REAL NOT NULL DEFAULT 3.5,
			duration_years INTEGER NOT NULL DEFAULT 20,
			insurance_rate REAL NOT NULL DEFAULT 0.34,
			notary_rate REAL NOT NULL DEFAULT 7.5,
			agency_rate REAL NOT NULL DEFAULT 5.0,
			agency_fixed REAL NOT NULL DEFAULT 0,
			bank_fees REAL NOT NULL DEFAULT 0,
			guarantee_fees REAL NOT NULL DEFAULT 0,
			broker_fees REAL NOT NULL DEFAULT 0,
			start_year INTEGER NOT NULL DEFAULT 2026,
			start_month INTEGER NOT NULL DEFAULT 1,
			net_income_1 REAL NOT NULL DEFAULT 0,
			net_income_2 REAL NOT NULL DEFAULT 0,
			monthly_rent REAL NOT NULL DEFAULT 0,
			rent_increase_rate REAL NOT NULL DEFAULT 2.0,
			savings_rate REAL NOT NULL DEFAULT 4.0,
			inflation_rate REAL NOT NULL DEFAULT 2.0,
			property_tax REAL NOT NULL DEFAULT 0,
			condo_fees REAL NOT NULL DEFAULT 0,
			maintenance_rate REAL NOT NULL DEFAULT 1.0,
			renovation_cost REAL NOT NULL DEFAULT 0,
			renovation_value_rate REAL NOT NULL DEFAULT 70.0,
			down_payment_1 REAL NOT NULL DEFAULT 0,
			down_payment_2 REAL NOT NULL DEFAULT 0,
			payment_split_mode TEXT NOT NULL DEFAULT 'prorata',
			current_sale_price REAL NOT NULL DEFAULT 0,
			current_loan_balance REAL NOT NULL DEFAULT 0,
			current_loan_rate REAL NOT NULL DEFAULT 0,
			current_loan_lines TEXT NOT NULL DEFAULT '[]',
			current_original_loan REAL NOT NULL DEFAULT 0,
			current_down_payment_1 REAL NOT NULL DEFAULT 0,
			current_renovation_cost REAL NOT NULL DEFAULT 0,
			current_renovation_share_2 REAL NOT NULL DEFAULT 0,
			early_repayment_penalty REAL NOT NULL DEFAULT 0,
			sale_property_share_1 REAL NOT NULL DEFAULT 50,
			virtual_contribution_2 REAL NOT NULL DEFAULT 0,
			virtual_profit_share_2 REAL NOT NULL DEFAULT 0,
			rfr_year_2_1 REAL NOT NULL DEFAULT 0,
			rfr_year_1_1 REAL NOT NULL DEFAULT 0,
			rfr_year_2_2 REAL NOT NULL DEFAULT 0,
			rfr_year_1_2 REAL NOT NULL DEFAULT 0,
			household_size INTEGER NOT NULL DEFAULT 1,
			property_zone TEXT NOT NULL DEFAULT 'B1',
			new_loan_lines TEXT NOT NULL DEFAULT '[]',
			work_lines TEXT NOT NULL DEFAULT '[]'
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Run migrations for any missing columns
	runMigrations(db)

	return &SQLiteStore{db: db}, nil
}

// runMigrations adds any missing columns to the table.
// With sqlx, order doesn't matter - just add new columns here.
func runMigrations(db *sqlx.DB) {
	migrations := []string{
		`ALTER TABLE simulation_inputs ADD COLUMN savings_rate REAL NOT NULL DEFAULT 4.0`,
		`ALTER TABLE simulation_inputs ADD COLUMN renovation_value_rate REAL NOT NULL DEFAULT 70.0`,
		`ALTER TABLE simulation_inputs ADD COLUMN down_payment_1 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN down_payment_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN payment_split_mode TEXT NOT NULL DEFAULT 'prorata'`,
		`ALTER TABLE simulation_inputs ADD COLUMN property_tax REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN condo_fees REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN renovation_cost REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_sale_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_loan_balance REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN early_repayment_penalty REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN sale_property_share_1 REAL NOT NULL DEFAULT 50`,
		`ALTER TABLE simulation_inputs ADD COLUMN virtual_contribution_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_loan_rate REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN virtual_profit_share_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_original_loan REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_down_payment_1 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_renovation_cost REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_renovation_share_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_loan_lines TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE simulation_inputs ADD COLUMN rfr_year_2_1 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN rfr_year_1_1 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN rfr_year_2_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN rfr_year_1_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN household_size INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE simulation_inputs ADD COLUMN property_zone TEXT NOT NULL DEFAULT 'B1'`,
		`ALTER TABLE simulation_inputs ADD COLUMN new_loan_lines TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE simulation_inputs ADD COLUMN guarantee_fees REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN inflation_rate REAL NOT NULL DEFAULT 2.0`,
		`ALTER TABLE simulation_inputs ADD COLUMN work_lines TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE simulation_inputs ADD COLUMN maintenance_rate REAL NOT NULL DEFAULT 1.0`,
		`ALTER TABLE simulation_inputs ADD COLUMN virtual_monthly_payment_2 REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN broker_fees REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_target_payment REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_loan_start_year INTEGER NOT NULL DEFAULT 2020`,
		`ALTER TABLE simulation_inputs ADD COLUMN current_loan_start_month INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE simulation_inputs ADD COLUMN virtual_payment_tiers_2 TEXT NOT NULL DEFAULT '[]'`,
	}

	for _, migration := range migrations {
		// Ignore errors - column may already exist
		_, _ = db.Exec(migration)
	}
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
	inputs := &FormInputs{}

	// Select explicit columns to avoid errors from stale columns in the DB
	err := s.db.Get(inputs, `SELECT id, property_price, down_payment, interest_rate, duration_years,
		insurance_rate, notary_rate, agency_rate, agency_fixed, bank_fees, guarantee_fees, broker_fees,
		start_year, start_month, net_income_1, net_income_2, monthly_rent, rent_increase_rate,
		savings_rate, inflation_rate, property_tax, condo_fees, maintenance_rate,
		renovation_cost, renovation_value_rate, down_payment_1, down_payment_2,
		payment_split_mode, current_sale_price, current_loan_balance, current_loan_rate,
		current_loan_lines, current_loan_start_year, current_loan_start_month,
		current_original_loan, current_down_payment_1,
		current_renovation_cost, current_renovation_share_2, early_repayment_penalty,
		sale_property_share_1, virtual_contribution_2, virtual_profit_share_2,
		virtual_monthly_payment_2, virtual_payment_tiers_2, rfr_year_2_1, rfr_year_1_1, rfr_year_2_2, rfr_year_1_2,
		household_size, property_zone, new_loan_lines, work_lines
		FROM simulation_inputs WHERE id = 1`)

	if err == sql.ErrNoRows {
		return DefaultInputs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load inputs: %w", err)
	}

	// Auto-migration: if current_loan_lines is empty but old fields have values,
	// create a loan line from the old data
	if (inputs.CurrentLoanLines == "" || inputs.CurrentLoanLines == "[]") && inputs.CurrentLoanBalance > 0 {
		inputs.CurrentLoanLines = fmt.Sprintf(`[{"label":"Prêt principal","balance":%.2f,"rate":%.2f,"ira":0}]`,
			inputs.CurrentLoanBalance, inputs.CurrentLoanRate)
	}

	return inputs, nil
}

// Save persists the form inputs to the database.
func (s *SQLiteStore) Save(inputs *FormInputs) error {
	inputs.ID = 1 // Ensure ID is always 1

	_, err := s.db.NamedExec(`
		INSERT INTO simulation_inputs (
			id, property_price, down_payment, interest_rate, duration_years,
			insurance_rate, notary_rate, agency_rate, agency_fixed,
			bank_fees, guarantee_fees, broker_fees, start_year, start_month,
			net_income_1, net_income_2, monthly_rent, rent_increase_rate,
			savings_rate, inflation_rate, property_tax, condo_fees, maintenance_rate,
			renovation_cost, renovation_value_rate, down_payment_1, down_payment_2,
			payment_split_mode, current_sale_price, current_loan_balance,
			current_loan_rate, current_loan_lines, current_loan_start_year, current_loan_start_month,
			current_original_loan, current_down_payment_1, current_renovation_cost,
			current_renovation_share_2, early_repayment_penalty,
			sale_property_share_1, virtual_contribution_2,
			virtual_profit_share_2, virtual_monthly_payment_2, virtual_payment_tiers_2,
			rfr_year_2_1, rfr_year_1_1,
			rfr_year_2_2, rfr_year_1_2, household_size, property_zone,
			new_loan_lines, work_lines
		) VALUES (
			:id, :property_price, :down_payment, :interest_rate, :duration_years,
			:insurance_rate, :notary_rate, :agency_rate, :agency_fixed,
			:bank_fees, :guarantee_fees, :broker_fees, :start_year, :start_month,
			:net_income_1, :net_income_2, :monthly_rent, :rent_increase_rate,
			:savings_rate, :inflation_rate, :property_tax, :condo_fees, :maintenance_rate,
			:renovation_cost, :renovation_value_rate, :down_payment_1, :down_payment_2,
			:payment_split_mode, :current_sale_price, :current_loan_balance,
			:current_loan_rate, :current_loan_lines, :current_loan_start_year, :current_loan_start_month,
			:current_original_loan, :current_down_payment_1, :current_renovation_cost,
			:current_renovation_share_2, :early_repayment_penalty,
			:sale_property_share_1, :virtual_contribution_2,
			:virtual_profit_share_2, :virtual_monthly_payment_2, :virtual_payment_tiers_2,
			:rfr_year_2_1, :rfr_year_1_1,
			:rfr_year_2_2, :rfr_year_1_2, :household_size, :property_zone,
			:new_loan_lines, :work_lines
		)
		ON CONFLICT(id) DO UPDATE SET
			property_price = :property_price,
			down_payment = :down_payment,
			interest_rate = :interest_rate,
			duration_years = :duration_years,
			insurance_rate = :insurance_rate,
			notary_rate = :notary_rate,
			agency_rate = :agency_rate,
			agency_fixed = :agency_fixed,
			bank_fees = :bank_fees,
			guarantee_fees = :guarantee_fees,
			broker_fees = :broker_fees,
			start_year = :start_year,
			start_month = :start_month,
			net_income_1 = :net_income_1,
			net_income_2 = :net_income_2,
			monthly_rent = :monthly_rent,
			rent_increase_rate = :rent_increase_rate,
			savings_rate = :savings_rate,
			inflation_rate = :inflation_rate,
			property_tax = :property_tax,
			condo_fees = :condo_fees,
			maintenance_rate = :maintenance_rate,
			renovation_cost = :renovation_cost,
			renovation_value_rate = :renovation_value_rate,
			down_payment_1 = :down_payment_1,
			down_payment_2 = :down_payment_2,
			payment_split_mode = :payment_split_mode,
			current_sale_price = :current_sale_price,
			current_loan_balance = :current_loan_balance,
			current_loan_rate = :current_loan_rate,
			current_loan_lines = :current_loan_lines,
			current_loan_start_year = :current_loan_start_year,
			current_loan_start_month = :current_loan_start_month,
			current_original_loan = :current_original_loan,
			current_down_payment_1 = :current_down_payment_1,
			current_renovation_cost = :current_renovation_cost,
			current_renovation_share_2 = :current_renovation_share_2,
			early_repayment_penalty = :early_repayment_penalty,
			sale_property_share_1 = :sale_property_share_1,
			virtual_contribution_2 = :virtual_contribution_2,
			virtual_profit_share_2 = :virtual_profit_share_2,
			virtual_monthly_payment_2 = :virtual_monthly_payment_2,
			virtual_payment_tiers_2 = :virtual_payment_tiers_2,
			rfr_year_2_1 = :rfr_year_2_1,
			rfr_year_1_1 = :rfr_year_1_1,
			rfr_year_2_2 = :rfr_year_2_2,
			rfr_year_1_2 = :rfr_year_1_2,
			household_size = :household_size,
			property_zone = :property_zone,
			new_loan_lines = :new_loan_lines,
			work_lines = :work_lines
	`, inputs)

	if err != nil {
		return fmt.Errorf("failed to save inputs: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
