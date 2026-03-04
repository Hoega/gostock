package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	// Create stock_positions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stock_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			isin TEXT NOT NULL DEFAULT '',
			broker TEXT NOT NULL,
			quantity REAL NOT NULL,
			purchase_price REAL NOT NULL,
			current_price REAL NOT NULL,
			purchase_fees REAL NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'EUR',
			sector TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create stock_positions table: %w", err)
	}

	// Create crypto_positions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS crypto_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol TEXT NOT NULL,
			coingecko_id TEXT NOT NULL,
			name TEXT NOT NULL,
			wallet TEXT NOT NULL,
			quantity REAL NOT NULL,
			purchase_price REAL NOT NULL,
			current_price REAL NOT NULL,
			purchase_fees REAL NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create crypto_positions table: %w", err)
	}

	// Create stock_sales table for tax reporting (2042-C)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stock_sales (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			isin TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			broker TEXT NOT NULL,
			purchase_date TEXT NOT NULL,
			purchase_price REAL NOT NULL,
			purchase_fees REAL NOT NULL DEFAULT 0,
			sale_date TEXT NOT NULL,
			sale_price REAL NOT NULL,
			sale_fees REAL NOT NULL DEFAULT 0,
			quantity REAL NOT NULL,
			currency TEXT NOT NULL DEFAULT 'EUR',
			tax_year INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create stock_sales table: %w", err)
	}

	// Create crypto_sales table for tax reporting (2086)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS crypto_sales (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol TEXT NOT NULL,
			name TEXT NOT NULL,
			wallet TEXT NOT NULL,
			purchase_date TEXT NOT NULL,
			purchase_price REAL NOT NULL,
			purchase_fees REAL NOT NULL DEFAULT 0,
			sale_date TEXT NOT NULL,
			sale_price REAL NOT NULL,
			sale_fees REAL NOT NULL DEFAULT 0,
			quantity REAL NOT NULL,
			portfolio_value_at_sale REAL NOT NULL DEFAULT 0,
			portfolio_acquisition_cost REAL NOT NULL DEFAULT 0,
			tax_year INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create crypto_sales table: %w", err)
	}

	// Create stock_purchases table for PRU calculation
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stock_purchases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			isin TEXT NOT NULL,
			name TEXT NOT NULL,
			broker TEXT NOT NULL,
			quantity REAL NOT NULL,
			unit_price REAL NOT NULL,
			fees REAL NOT NULL DEFAULT 0,
			purchase_date TEXT NOT NULL,
			currency TEXT NOT NULL DEFAULT 'EUR',
			remaining_quantity REAL NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create stock_purchases table: %w", err)
	}

	// Create cash_positions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cash_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bank_name TEXT NOT NULL,
			amount REAL NOT NULL,
			account_type TEXT NOT NULL,
			interest_rate REAL NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create cash_positions table: %w", err)
	}

	// Create watchlist_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS watchlist_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			isin TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create watchlist_items table: %w", err)
	}

	// Create portfolio_snapshots table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS portfolio_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			stock_total REAL NOT NULL DEFAULT 0,
			crypto_total REAL NOT NULL DEFAULT 0,
			cash_total REAL NOT NULL DEFAULT 0,
			real_estate_total REAL NOT NULL DEFAULT 0,
			grand_total REAL NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create portfolio_snapshots table: %w", err)
	}

	// Create budget_inputs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS budget_inputs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			gross_salary REAL NOT NULL DEFAULT 0,
			net_salary REAL NOT NULL DEFAULT 0,
			dividends REAL NOT NULL DEFAULT 0,
			rental_income REAL NOT NULL DEFAULT 0,
			other_income REAL NOT NULL DEFAULT 0,
			income_tax REAL NOT NULL DEFAULT 0,
			housing REAL NOT NULL DEFAULT 0,
			lifestyle REAL NOT NULL DEFAULT 0,
			transport REAL NOT NULL DEFAULT 0,
			insurance REAL NOT NULL DEFAULT 0,
			subscriptions REAL NOT NULL DEFAULT 0,
			other_expenses REAL NOT NULL DEFAULT 0,
			pea REAL NOT NULL DEFAULT 0,
			assurance_vie REAL NOT NULL DEFAULT 0,
			per REAL NOT NULL DEFAULT 0,
			livret_a REAL NOT NULL DEFAULT 0,
			crypto_savings REAL NOT NULL DEFAULT 0,
			other_savings REAL NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create budget_inputs table: %w", err)
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
		// Energy comparison fields
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_gas REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_electricity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_gas_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_electricity_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_other REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_other_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_gas REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_electricity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_gas_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_electricity_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_other REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_other_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_surface REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_surface REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_price_increase REAL NOT NULL DEFAULT 4.0`,
		// DPE fields
		`ALTER TABLE simulation_inputs ADD COLUMN energy_1_dpe REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_2_dpe REAL NOT NULL DEFAULT 0`,
		// Energy3 fields
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_gas REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_electricity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_gas_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_electricity_kwh REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_other REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_other_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_surface REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN energy_3_dpe REAL NOT NULL DEFAULT 0`,
		// Resale projection fields
		`ALTER TABLE simulation_inputs ADD COLUMN resale_rates TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE simulation_inputs ADD COLUMN resale_sell_costs REAL NOT NULL DEFAULT 0`,
		// Budget migrations
		// Bridge loan fields
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_enabled INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_quotity REAL NOT NULL DEFAULT 70`,
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_rate REAL NOT NULL DEFAULT 3.5`,
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_duration INTEGER NOT NULL DEFAULT 12`,
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_insurance REAL NOT NULL DEFAULT 0.34`,
		`ALTER TABLE simulation_inputs ADD COLUMN bridge_loan_franchise TEXT NOT NULL DEFAULT 'partielle'`,
		// Budget migrations
		`ALTER TABLE budget_inputs ADD COLUMN childcare REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE budget_inputs ADD COLUMN subscriptions_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE budget_inputs ADD COLUMN meal_vouchers REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE budget_inputs ADD COLUMN lifestyle_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE budget_inputs ADD COLUMN other_expenses_json TEXT NOT NULL DEFAULT '[]'`,
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
		household_size, property_zone, new_loan_lines, work_lines,
		energy_1_gas, energy_1_electricity, energy_1_gas_kwh, energy_1_electricity_kwh,
		energy_1_other, energy_1_other_label, energy_1_label, energy_1_surface, energy_1_dpe,
		energy_2_gas, energy_2_electricity, energy_2_gas_kwh, energy_2_electricity_kwh,
		energy_2_other, energy_2_other_label, energy_2_label, energy_2_surface, energy_2_dpe,
		energy_3_gas, energy_3_electricity, energy_3_gas_kwh, energy_3_electricity_kwh,
		energy_3_other, energy_3_other_label, energy_3_label, energy_3_surface, energy_3_dpe, energy_price_increase,
		resale_rates, resale_sell_costs,
		bridge_loan_enabled, bridge_loan_quotity, bridge_loan_rate,
		bridge_loan_duration, bridge_loan_insurance, bridge_loan_franchise
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
			new_loan_lines, work_lines,
			energy_1_gas, energy_1_electricity, energy_1_gas_kwh, energy_1_electricity_kwh,
			energy_1_other, energy_1_other_label, energy_1_label, energy_1_surface, energy_1_dpe,
			energy_2_gas, energy_2_electricity, energy_2_gas_kwh, energy_2_electricity_kwh,
			energy_2_other, energy_2_other_label, energy_2_label, energy_2_surface, energy_2_dpe,
			energy_3_gas, energy_3_electricity, energy_3_gas_kwh, energy_3_electricity_kwh,
			energy_3_other, energy_3_other_label, energy_3_label, energy_3_surface, energy_3_dpe, energy_price_increase,
			resale_rates, resale_sell_costs,
			bridge_loan_enabled, bridge_loan_quotity, bridge_loan_rate,
			bridge_loan_duration, bridge_loan_insurance, bridge_loan_franchise
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
			:new_loan_lines, :work_lines,
			:energy_1_gas, :energy_1_electricity, :energy_1_gas_kwh, :energy_1_electricity_kwh,
			:energy_1_other, :energy_1_other_label, :energy_1_label, :energy_1_surface, :energy_1_dpe,
			:energy_2_gas, :energy_2_electricity, :energy_2_gas_kwh, :energy_2_electricity_kwh,
			:energy_2_other, :energy_2_other_label, :energy_2_label, :energy_2_surface, :energy_2_dpe,
			:energy_3_gas, :energy_3_electricity, :energy_3_gas_kwh, :energy_3_electricity_kwh,
			:energy_3_other, :energy_3_other_label, :energy_3_label, :energy_3_surface, :energy_3_dpe, :energy_price_increase,
			:resale_rates, :resale_sell_costs,
			:bridge_loan_enabled, :bridge_loan_quotity, :bridge_loan_rate,
			:bridge_loan_duration, :bridge_loan_insurance, :bridge_loan_franchise
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
			work_lines = :work_lines,
			energy_1_gas = :energy_1_gas,
			energy_1_electricity = :energy_1_electricity,
			energy_1_gas_kwh = :energy_1_gas_kwh,
			energy_1_electricity_kwh = :energy_1_electricity_kwh,
			energy_1_other = :energy_1_other,
			energy_1_other_label = :energy_1_other_label,
			energy_1_label = :energy_1_label,
			energy_1_surface = :energy_1_surface,
			energy_1_dpe = :energy_1_dpe,
			energy_2_gas = :energy_2_gas,
			energy_2_electricity = :energy_2_electricity,
			energy_2_gas_kwh = :energy_2_gas_kwh,
			energy_2_electricity_kwh = :energy_2_electricity_kwh,
			energy_2_other = :energy_2_other,
			energy_2_other_label = :energy_2_other_label,
			energy_2_label = :energy_2_label,
			energy_2_surface = :energy_2_surface,
			energy_2_dpe = :energy_2_dpe,
			energy_3_gas = :energy_3_gas,
			energy_3_electricity = :energy_3_electricity,
			energy_3_gas_kwh = :energy_3_gas_kwh,
			energy_3_electricity_kwh = :energy_3_electricity_kwh,
			energy_3_other = :energy_3_other,
			energy_3_other_label = :energy_3_other_label,
			energy_3_label = :energy_3_label,
			energy_3_surface = :energy_3_surface,
			energy_3_dpe = :energy_3_dpe,
			energy_price_increase = :energy_price_increase,
			resale_rates = :resale_rates,
			resale_sell_costs = :resale_sell_costs,
			bridge_loan_enabled = :bridge_loan_enabled,
			bridge_loan_quotity = :bridge_loan_quotity,
			bridge_loan_rate = :bridge_loan_rate,
			bridge_loan_duration = :bridge_loan_duration,
			bridge_loan_insurance = :bridge_loan_insurance,
			bridge_loan_franchise = :bridge_loan_franchise
	`, inputs)

	if err != nil {
		return fmt.Errorf("failed to save inputs: %w", err)
	}
	return nil
}

// LoadPositions retrieves all stock positions ordered by broker then name.
func (s *SQLiteStore) LoadPositions() ([]StockPosition, error) {
	var positions []StockPosition
	err := s.db.Select(&positions, `SELECT id, name, isin, broker, quantity, purchase_price, current_price, purchase_fees, currency, sector FROM stock_positions ORDER BY broker, name`)
	if err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}
	return positions, nil
}

// SavePosition inserts or updates a stock position.
func (s *SQLiteStore) SavePosition(pos *StockPosition) error {
	if pos.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO stock_positions (name, isin, broker, quantity, purchase_price, current_price, purchase_fees, currency, sector)
			VALUES (:name, :isin, :broker, :quantity, :purchase_price, :current_price, :purchase_fees, :currency, :sector)
		`, pos)
		if err != nil {
			return fmt.Errorf("failed to insert position: %w", err)
		}
		id, _ := result.LastInsertId()
		pos.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE stock_positions SET
			name = :name, isin = :isin, broker = :broker, quantity = :quantity,
			purchase_price = :purchase_price, current_price = :current_price,
			purchase_fees = :purchase_fees, currency = :currency, sector = :sector
		WHERE id = :id
	`, pos)
	if err != nil {
		return fmt.Errorf("failed to update position: %w", err)
	}
	return nil
}

// DeletePosition removes a stock position by ID.
func (s *SQLiteStore) DeletePosition(id int) error {
	_, err := s.db.Exec(`DELETE FROM stock_positions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete position: %w", err)
	}
	return nil
}

// LoadCryptoPositions retrieves all crypto positions ordered by wallet then symbol.
func (s *SQLiteStore) LoadCryptoPositions() ([]CryptoPosition, error) {
	var positions []CryptoPosition
	err := s.db.Select(&positions, `SELECT id, symbol, coingecko_id, name, wallet, quantity, purchase_price, current_price, purchase_fees FROM crypto_positions ORDER BY wallet, symbol`)
	if err != nil {
		return nil, fmt.Errorf("failed to load crypto positions: %w", err)
	}
	return positions, nil
}

// SaveCryptoPosition inserts or updates a crypto position.
func (s *SQLiteStore) SaveCryptoPosition(pos *CryptoPosition) error {
	if pos.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO crypto_positions (symbol, coingecko_id, name, wallet, quantity, purchase_price, current_price, purchase_fees)
			VALUES (:symbol, :coingecko_id, :name, :wallet, :quantity, :purchase_price, :current_price, :purchase_fees)
		`, pos)
		if err != nil {
			return fmt.Errorf("failed to insert crypto position: %w", err)
		}
		id, _ := result.LastInsertId()
		pos.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE crypto_positions SET
			symbol = :symbol, coingecko_id = :coingecko_id, name = :name, wallet = :wallet,
			quantity = :quantity, purchase_price = :purchase_price, current_price = :current_price,
			purchase_fees = :purchase_fees
		WHERE id = :id
	`, pos)
	if err != nil {
		return fmt.Errorf("failed to update crypto position: %w", err)
	}
	return nil
}

// DeleteCryptoPosition removes a crypto position by ID.
func (s *SQLiteStore) DeleteCryptoPosition(id int) error {
	_, err := s.db.Exec(`DELETE FROM crypto_positions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete crypto position: %w", err)
	}
	return nil
}

// LoadStockSales retrieves all stock sales ordered by tax year descending, then sale date.
func (s *SQLiteStore) LoadStockSales() ([]StockSale, error) {
	var sales []StockSale
	err := s.db.Select(&sales, `SELECT id, isin, name, broker, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, currency, tax_year FROM stock_sales ORDER BY tax_year DESC, sale_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to load stock sales: %w", err)
	}
	return sales, nil
}

// LoadStockSalesByYear retrieves stock sales for a specific tax year.
func (s *SQLiteStore) LoadStockSalesByYear(year int) ([]StockSale, error) {
	var sales []StockSale
	err := s.db.Select(&sales, `SELECT id, isin, name, broker, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, currency, tax_year FROM stock_sales WHERE tax_year = ? ORDER BY sale_date DESC`, year)
	if err != nil {
		return nil, fmt.Errorf("failed to load stock sales for year %d: %w", year, err)
	}
	return sales, nil
}

// SaveStockSale inserts or updates a stock sale.
func (s *SQLiteStore) SaveStockSale(sale *StockSale) error {
	if sale.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO stock_sales (isin, name, broker, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, currency, tax_year)
			VALUES (:isin, :name, :broker, :purchase_date, :purchase_price, :purchase_fees, :sale_date, :sale_price, :sale_fees, :quantity, :currency, :tax_year)
		`, sale)
		if err != nil {
			return fmt.Errorf("failed to insert stock sale: %w", err)
		}
		id, _ := result.LastInsertId()
		sale.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE stock_sales SET
			isin = :isin, name = :name, broker = :broker, purchase_date = :purchase_date,
			purchase_price = :purchase_price, purchase_fees = :purchase_fees,
			sale_date = :sale_date, sale_price = :sale_price, sale_fees = :sale_fees,
			quantity = :quantity, currency = :currency, tax_year = :tax_year
		WHERE id = :id
	`, sale)
	if err != nil {
		return fmt.Errorf("failed to update stock sale: %w", err)
	}
	return nil
}

// DeleteStockSale removes a stock sale by ID.
func (s *SQLiteStore) DeleteStockSale(id int) error {
	_, err := s.db.Exec(`DELETE FROM stock_sales WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete stock sale: %w", err)
	}
	return nil
}

// LoadCryptoSales retrieves all crypto sales ordered by tax year descending, then sale date.
func (s *SQLiteStore) LoadCryptoSales() ([]CryptoSale, error) {
	var sales []CryptoSale
	err := s.db.Select(&sales, `SELECT id, symbol, name, wallet, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, portfolio_value_at_sale, portfolio_acquisition_cost, tax_year FROM crypto_sales ORDER BY tax_year DESC, sale_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to load crypto sales: %w", err)
	}
	return sales, nil
}

// LoadCryptoSalesByYear retrieves crypto sales for a specific tax year.
func (s *SQLiteStore) LoadCryptoSalesByYear(year int) ([]CryptoSale, error) {
	var sales []CryptoSale
	err := s.db.Select(&sales, `SELECT id, symbol, name, wallet, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, portfolio_value_at_sale, portfolio_acquisition_cost, tax_year FROM crypto_sales WHERE tax_year = ? ORDER BY sale_date DESC`, year)
	if err != nil {
		return nil, fmt.Errorf("failed to load crypto sales for year %d: %w", year, err)
	}
	return sales, nil
}

// SaveCryptoSale inserts or updates a crypto sale.
func (s *SQLiteStore) SaveCryptoSale(sale *CryptoSale) error {
	if sale.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO crypto_sales (symbol, name, wallet, purchase_date, purchase_price, purchase_fees, sale_date, sale_price, sale_fees, quantity, portfolio_value_at_sale, portfolio_acquisition_cost, tax_year)
			VALUES (:symbol, :name, :wallet, :purchase_date, :purchase_price, :purchase_fees, :sale_date, :sale_price, :sale_fees, :quantity, :portfolio_value_at_sale, :portfolio_acquisition_cost, :tax_year)
		`, sale)
		if err != nil {
			return fmt.Errorf("failed to insert crypto sale: %w", err)
		}
		id, _ := result.LastInsertId()
		sale.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE crypto_sales SET
			symbol = :symbol, name = :name, wallet = :wallet, purchase_date = :purchase_date,
			purchase_price = :purchase_price, purchase_fees = :purchase_fees,
			sale_date = :sale_date, sale_price = :sale_price, sale_fees = :sale_fees,
			quantity = :quantity, portfolio_value_at_sale = :portfolio_value_at_sale,
			portfolio_acquisition_cost = :portfolio_acquisition_cost, tax_year = :tax_year
		WHERE id = :id
	`, sale)
	if err != nil {
		return fmt.Errorf("failed to update crypto sale: %w", err)
	}
	return nil
}

// DeleteCryptoSale removes a crypto sale by ID.
func (s *SQLiteStore) DeleteCryptoSale(id int) error {
	_, err := s.db.Exec(`DELETE FROM crypto_sales WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete crypto sale: %w", err)
	}
	return nil
}

// GetTaxYears returns all distinct tax years from both stock and crypto sales.
func (s *SQLiteStore) GetTaxYears() ([]int, error) {
	var years []int
	err := s.db.Select(&years, `
		SELECT DISTINCT tax_year FROM (
			SELECT tax_year FROM stock_sales
			UNION
			SELECT tax_year FROM crypto_sales
		) ORDER BY tax_year DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get tax years: %w", err)
	}
	return years, nil
}

// LoadStockPurchases retrieves all stock purchases ordered by purchase date.
func (s *SQLiteStore) LoadStockPurchases() ([]StockPurchase, error) {
	var purchases []StockPurchase
	err := s.db.Select(&purchases, `SELECT id, isin, name, broker, quantity, unit_price, fees, purchase_date, currency, remaining_quantity FROM stock_purchases ORDER BY purchase_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to load stock purchases: %w", err)
	}
	return purchases, nil
}

// LoadStockPurchasesByISIN retrieves stock purchases for a specific ISIN ordered by purchase date (oldest first for FIFO).
func (s *SQLiteStore) LoadStockPurchasesByISIN(isin string) ([]StockPurchase, error) {
	var purchases []StockPurchase
	err := s.db.Select(&purchases, `SELECT id, isin, name, broker, quantity, unit_price, fees, purchase_date, currency, remaining_quantity FROM stock_purchases WHERE isin = ? AND remaining_quantity > 0 ORDER BY purchase_date ASC`, isin)
	if err != nil {
		return nil, fmt.Errorf("failed to load stock purchases for ISIN %s: %w", isin, err)
	}
	return purchases, nil
}

// SaveStockPurchase inserts or updates a stock purchase.
func (s *SQLiteStore) SaveStockPurchase(purchase *StockPurchase) error {
	if purchase.ID == 0 {
		// New purchase: remaining_quantity = quantity
		purchase.RemainingQuantity = purchase.Quantity
		result, err := s.db.NamedExec(`
			INSERT INTO stock_purchases (isin, name, broker, quantity, unit_price, fees, purchase_date, currency, remaining_quantity)
			VALUES (:isin, :name, :broker, :quantity, :unit_price, :fees, :purchase_date, :currency, :remaining_quantity)
		`, purchase)
		if err != nil {
			return fmt.Errorf("failed to insert stock purchase: %w", err)
		}
		id, _ := result.LastInsertId()
		purchase.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE stock_purchases SET
			isin = :isin, name = :name, broker = :broker, quantity = :quantity,
			unit_price = :unit_price, fees = :fees, purchase_date = :purchase_date,
			currency = :currency, remaining_quantity = :remaining_quantity
		WHERE id = :id
	`, purchase)
	if err != nil {
		return fmt.Errorf("failed to update stock purchase: %w", err)
	}
	return nil
}

// DeleteStockPurchase removes a stock purchase by ID.
func (s *SQLiteStore) DeleteStockPurchase(id int) error {
	_, err := s.db.Exec(`DELETE FROM stock_purchases WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete stock purchase: %w", err)
	}
	return nil
}

// CalculatePRUByISIN calculates the weighted average PRU for an ISIN based on remaining quantities.
// Formula: PRU = Σ(remaining_quantity × unit_price + proportional_fees) / Σ(remaining_quantity)
func (s *SQLiteStore) CalculatePRUByISIN(isin string) (float64, error) {
	var result struct {
		TotalCost     float64 `db:"total_cost"`
		TotalQuantity float64 `db:"total_quantity"`
	}
	err := s.db.Get(&result, `
		SELECT
			COALESCE(SUM(remaining_quantity * unit_price + (fees * remaining_quantity / quantity)), 0) as total_cost,
			COALESCE(SUM(remaining_quantity), 0) as total_quantity
		FROM stock_purchases
		WHERE isin = ? AND remaining_quantity > 0
	`, isin)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate PRU for ISIN %s: %w", isin, err)
	}
	if result.TotalQuantity == 0 {
		return 0, nil
	}
	return result.TotalCost / result.TotalQuantity, nil
}

// GetAvailableQuantityByISIN returns the total remaining quantity for an ISIN.
func (s *SQLiteStore) GetAvailableQuantityByISIN(isin string) (float64, error) {
	var qty float64
	err := s.db.Get(&qty, `SELECT COALESCE(SUM(remaining_quantity), 0) FROM stock_purchases WHERE isin = ? AND remaining_quantity > 0`, isin)
	if err != nil {
		return 0, fmt.Errorf("failed to get available quantity for ISIN %s: %w", isin, err)
	}
	return qty, nil
}

// ReduceRemainingQuantity reduces remaining quantities using FIFO (oldest purchases first).
func (s *SQLiteStore) ReduceRemainingQuantity(isin string, qty float64) error {
	purchases, err := s.LoadStockPurchasesByISIN(isin)
	if err != nil {
		return err
	}

	remaining := qty
	for _, p := range purchases {
		if remaining <= 0 {
			break
		}
		reduction := remaining
		if reduction > p.RemainingQuantity {
			reduction = p.RemainingQuantity
		}
		newRemaining := p.RemainingQuantity - reduction
		_, err := s.db.Exec(`UPDATE stock_purchases SET remaining_quantity = ? WHERE id = ?`, newRemaining, p.ID)
		if err != nil {
			return fmt.Errorf("failed to reduce remaining quantity: %w", err)
		}
		remaining -= reduction
	}
	return nil
}

// GetStockPurchaseNameByISIN returns the name and broker for an ISIN from any purchase (ignoring remaining_quantity).
func (s *SQLiteStore) GetStockPurchaseNameByISIN(isin string) (string, string, error) {
	var result struct {
		Name   string `db:"name"`
		Broker string `db:"broker"`
	}
	err := s.db.Get(&result, `SELECT name, broker FROM stock_purchases WHERE isin = ? LIMIT 1`, isin)
	if err != nil {
		return "", "", err
	}
	return result.Name, result.Broker, nil
}

// GetEarliestPurchaseDateByISIN returns the purchase date of the oldest lot with remaining stock (FIFO).
func (s *SQLiteStore) GetEarliestPurchaseDateByISIN(isin string) (string, error) {
	var date string
	err := s.db.Get(&date, `SELECT purchase_date FROM stock_purchases WHERE isin = ? AND remaining_quantity > 0 ORDER BY purchase_date ASC LIMIT 1`, isin)
	if err != nil {
		return "", err
	}
	return date, nil
}

// ResetRemainingQuantity resets the remaining quantity to the original quantity.
func (s *SQLiteStore) ResetRemainingQuantity(id int) error {
	_, err := s.db.Exec(`UPDATE stock_purchases SET remaining_quantity = quantity WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to reset remaining quantity: %w", err)
	}
	return nil
}

// LoadCashPositions retrieves all cash positions ordered by bank name then account type.
func (s *SQLiteStore) LoadCashPositions() ([]CashPosition, error) {
	var positions []CashPosition
	err := s.db.Select(&positions, `SELECT id, bank_name, amount, account_type, interest_rate FROM cash_positions ORDER BY bank_name, account_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to load cash positions: %w", err)
	}
	return positions, nil
}

// SaveCashPosition inserts or updates a cash position.
func (s *SQLiteStore) SaveCashPosition(pos *CashPosition) error {
	if pos.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO cash_positions (bank_name, amount, account_type, interest_rate)
			VALUES (:bank_name, :amount, :account_type, :interest_rate)
		`, pos)
		if err != nil {
			return fmt.Errorf("failed to insert cash position: %w", err)
		}
		id, _ := result.LastInsertId()
		pos.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE cash_positions SET
			bank_name = :bank_name, amount = :amount,
			account_type = :account_type, interest_rate = :interest_rate
		WHERE id = :id
	`, pos)
	if err != nil {
		return fmt.Errorf("failed to update cash position: %w", err)
	}
	return nil
}

// DeleteCashPosition removes a cash position by ID.
func (s *SQLiteStore) DeleteCashPosition(id int) error {
	_, err := s.db.Exec(`DELETE FROM cash_positions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete cash position: %w", err)
	}
	return nil
}

// LoadWatchlistItems retrieves all watchlist items ordered by name.
func (s *SQLiteStore) LoadWatchlistItems() ([]WatchlistItem, error) {
	var items []WatchlistItem
	err := s.db.Select(&items, `SELECT id, isin, name FROM watchlist_items ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to load watchlist items: %w", err)
	}
	return items, nil
}

// SaveWatchlistItem inserts or updates a watchlist item.
func (s *SQLiteStore) SaveWatchlistItem(item *WatchlistItem) error {
	if item.ID == 0 {
		result, err := s.db.NamedExec(`
			INSERT INTO watchlist_items (isin, name) VALUES (:isin, :name)
		`, item)
		if err != nil {
			return fmt.Errorf("failed to insert watchlist item: %w", err)
		}
		id, _ := result.LastInsertId()
		item.ID = int(id)
		return nil
	}

	_, err := s.db.NamedExec(`
		UPDATE watchlist_items SET isin = :isin, name = :name WHERE id = :id
	`, item)
	if err != nil {
		return fmt.Errorf("failed to update watchlist item: %w", err)
	}
	return nil
}

// DeleteWatchlistItem removes a watchlist item by ID.
func (s *SQLiteStore) DeleteWatchlistItem(id int) error {
	_, err := s.db.Exec(`DELETE FROM watchlist_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete watchlist item: %w", err)
	}
	return nil
}

// SavePortfolioSnapshot upserts a daily portfolio snapshot by date.
func (s *SQLiteStore) SavePortfolioSnapshot(snap *PortfolioSnapshot) error {
	_, err := s.db.NamedExec(`
		INSERT INTO portfolio_snapshots (date, stock_total, crypto_total, cash_total, real_estate_total, grand_total)
		VALUES (:date, :stock_total, :crypto_total, :cash_total, :real_estate_total, :grand_total)
		ON CONFLICT(date) DO UPDATE SET
			stock_total = :stock_total,
			crypto_total = :crypto_total,
			cash_total = :cash_total,
			real_estate_total = :real_estate_total,
			grand_total = :grand_total
	`, snap)
	if err != nil {
		return fmt.Errorf("failed to save portfolio snapshot: %w", err)
	}
	return nil
}

// LoadPortfolioSnapshots retrieves snapshots since the given date, ordered chronologically.
func (s *SQLiteStore) LoadPortfolioSnapshots(since time.Time) ([]PortfolioSnapshot, error) {
	var snapshots []PortfolioSnapshot
	err := s.db.Select(&snapshots, `SELECT id, date, stock_total, crypto_total, cash_total, real_estate_total, grand_total FROM portfolio_snapshots WHERE date >= ? ORDER BY date ASC`, since.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to load portfolio snapshots: %w", err)
	}
	return snapshots, nil
}

// LoadBudgetInputs retrieves the saved budget inputs or returns defaults.
func (s *SQLiteStore) LoadBudgetInputs() (*BudgetInputs, error) {
	inputs := &BudgetInputs{}
	err := s.db.Get(inputs, `SELECT id, gross_salary, net_salary, dividends, rental_income, other_income,
		income_tax, housing, lifestyle, transport, insurance, subscriptions, childcare, meal_vouchers, subscriptions_json, lifestyle_json, other_expenses, other_expenses_json,
		pea, assurance_vie, per, livret_a, crypto_savings, other_savings
		FROM budget_inputs WHERE id = 1`)
	if err == sql.ErrNoRows {
		return DefaultBudgetInputs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load budget inputs: %w", err)
	}
	return inputs, nil
}

// SaveBudgetInputs persists the budget inputs to the database.
func (s *SQLiteStore) SaveBudgetInputs(inputs *BudgetInputs) error {
	inputs.ID = 1
	_, err := s.db.NamedExec(`
		INSERT INTO budget_inputs (id, gross_salary, net_salary, dividends, rental_income, other_income,
			income_tax, housing, lifestyle, transport, insurance, subscriptions, childcare, meal_vouchers, subscriptions_json, lifestyle_json, other_expenses, other_expenses_json,
			pea, assurance_vie, per, livret_a, crypto_savings, other_savings)
		VALUES (:id, :gross_salary, :net_salary, :dividends, :rental_income, :other_income,
			:income_tax, :housing, :lifestyle, :transport, :insurance, :subscriptions, :childcare, :meal_vouchers, :subscriptions_json, :lifestyle_json, :other_expenses, :other_expenses_json,
			:pea, :assurance_vie, :per, :livret_a, :crypto_savings, :other_savings)
		ON CONFLICT(id) DO UPDATE SET
			gross_salary = :gross_salary, net_salary = :net_salary, dividends = :dividends,
			rental_income = :rental_income, other_income = :other_income, income_tax = :income_tax,
			housing = :housing, lifestyle = :lifestyle, transport = :transport,
			insurance = :insurance, subscriptions = :subscriptions, childcare = :childcare,
			meal_vouchers = :meal_vouchers, subscriptions_json = :subscriptions_json, lifestyle_json = :lifestyle_json,
			other_expenses = :other_expenses, other_expenses_json = :other_expenses_json,
			pea = :pea, assurance_vie = :assurance_vie, per = :per,
			livret_a = :livret_a, crypto_savings = :crypto_savings, other_savings = :other_savings
	`, inputs)
	if err != nil {
		return fmt.Errorf("failed to save budget inputs: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
