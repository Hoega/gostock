package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
)

type CreditHandler struct {
	templates *template.Template
	store     persistence.Store
}

func NewCreditHandler(templates *template.Template, store persistence.Store) *CreditHandler {
	return &CreditHandler{templates: templates, store: store}
}

// ShowForm renders the main credit simulator page.
func (h *CreditHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	inputs, err := h.store.Load()
	if err != nil {
		log.Printf("Failed to load saved inputs: %v", err)
		inputs = persistence.DefaultInputs()
	}

	if err := h.templates.ExecuteTemplate(w, "credit.html", inputs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Calculate processes the form and returns HTMX partials.
func (h *CreditHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	propertyPrice := parseFloat(r.FormValue("property_price"), 250000)
	downPayment1 := parseFloat(r.FormValue("down_payment_1"), 0)
	downPayment2 := parseFloat(r.FormValue("down_payment_2"), 0)
	paymentSplitMode := r.FormValue("payment_split_mode")
	if paymentSplitMode == "" {
		paymentSplitMode = "prorata"
	}

	// Property sale fields
	currentSalePrice := parseFloat(r.FormValue("current_sale_price"), 0)
	currentLoanLinesJSON := r.FormValue("current_loan_lines")
	currentOriginalLoan := parseFloat(r.FormValue("current_original_loan"), 0)
	currentDownPayment1 := parseFloat(r.FormValue("current_down_payment_1"), 0)
	currentRenovationCost := parseFloat(r.FormValue("current_renovation_cost"), 0)
	currentRenovationShare2 := parseFloat(r.FormValue("current_renovation_share_2"), 0)

	// Parse loan lines JSON
	var loanLines []model.LoanLine
	if currentLoanLinesJSON != "" && currentLoanLinesJSON != "[]" {
		if err := json.Unmarshal([]byte(currentLoanLinesJSON), &loanLines); err != nil {
			log.Printf("Failed to parse loan lines JSON: %v", err)
			loanLines = []model.LoanLine{}
		}
	}

	// Calculate totals from loan lines
	var currentLoanBalance float64
	var earlyRepaymentPenalty float64
	for _, line := range loanLines {
		currentLoanBalance += line.Balance
		earlyRepaymentPenalty += line.IRA
	}

	// Fallback to old fields if no loan lines but old fields have values
	if len(loanLines) == 0 {
		currentLoanBalance = parseFloat(r.FormValue("current_loan_balance"), 0)
		earlyRepaymentPenalty = parseFloat(r.FormValue("early_repayment_penalty"), 0)
	}
	salePropertyShare1 := parseFloat(r.FormValue("sale_property_share_1"), 50)
	virtualContribution2 := parseFloat(r.FormValue("virtual_contribution_2"), 0)
	virtualProfitShare2 := parseFloat(r.FormValue("virtual_profit_share_2"), 0)
	virtualMonthlyPayment2 := parseFloat(r.FormValue("virtual_monthly_payment_2"), 0)

	// Calculate sale proceeds
	saleProceeds := currentSalePrice - currentLoanBalance - earlyRepaymentPenalty
	if saleProceeds < 0 {
		saleProceeds = 0
	}

	// Total down payment includes sale proceeds
	downPayment := downPayment1 + downPayment2 + saleProceeds
	loanAmount := propertyPrice - downPayment
	if loanAmount < 0 {
		loanAmount = 0
	}
	if v := r.FormValue("loan_amount"); v != "" {
		loanAmount = parseFloat(v, loanAmount)
	}

	interestRate := parseFloat(r.FormValue("interest_rate"), 3.5)
	durationYears := parseInt(r.FormValue("duration_years"), 20)
	insuranceRate := parseFloat(r.FormValue("insurance_rate"), 0.34)
	notaryRate := parseFloat(r.FormValue("notary_rate"), 7.5)
	agencyRate := parseFloat(r.FormValue("agency_rate"), 5.0)
	agencyFixed := parseFloat(r.FormValue("agency_fixed"), 0)
	bankFees := parseFloat(r.FormValue("bank_fees"), 0)
	guaranteeFees := parseFloat(r.FormValue("guarantee_fees"), 0)
	startYear := parseInt(r.FormValue("start_year"), 0)
	startMonth := parseInt(r.FormValue("start_month"), 0)
	netIncome1 := parseFloat(r.FormValue("net_income_1"), 0)
	netIncome2 := parseFloat(r.FormValue("net_income_2"), 0)
	monthlyRent := parseFloat(r.FormValue("monthly_rent"), 0)
	rentIncreaseRate := parseFloat(r.FormValue("rent_increase_rate"), 2.0)
	savingsRate := parseFloat(r.FormValue("savings_rate"), 0)
	inflationRate := parseFloat(r.FormValue("inflation_rate"), 2.0)
	propertyTax := parseFloat(r.FormValue("property_tax"), 0)
	condoFees := parseFloat(r.FormValue("condo_fees"), 0)
	maintenanceRate := parseFloat(r.FormValue("maintenance_rate"), 1.0)
	renovationCost := parseFloat(r.FormValue("renovation_cost"), 0)
	renovationValueRate := parseFloat(r.FormValue("renovation_value_rate"), 70)

	// Parse work lines JSON (detailed work categories)
	workLinesJSON := r.FormValue("work_lines")
	var workLines []model.WorkLine
	if workLinesJSON != "" && workLinesJSON != "[]" {
		if err := json.Unmarshal([]byte(workLinesJSON), &workLines); err != nil {
			log.Printf("Failed to parse work lines JSON: %v", err)
			workLines = []model.WorkLine{}
		}
	}
	rfrYear2_1 := parseFloat(r.FormValue("rfr_year_2_1"), 0)
	rfrYear1_1 := parseFloat(r.FormValue("rfr_year_1_1"), 0)
	rfrYear2_2 := parseFloat(r.FormValue("rfr_year_2_2"), 0)
	rfrYear1_2 := parseFloat(r.FormValue("rfr_year_1_2"), 0)
	householdSize := parseInt(r.FormValue("household_size"), 1)
	propertyZone := r.FormValue("property_zone")
	if propertyZone == "" {
		propertyZone = "B1"
	}

	// Parse new loan lines JSON
	newLoanLinesJSON := r.FormValue("new_loan_lines")
	var newLoanLines []model.NewLoanLine
	if newLoanLinesJSON != "" && newLoanLinesJSON != "[]" {
		if err := json.Unmarshal([]byte(newLoanLinesJSON), &newLoanLines); err != nil {
			log.Printf("Failed to parse new loan lines JSON: %v", err)
			newLoanLines = []model.NewLoanLine{}
		}
	}

	// Save inputs to persistence
	formInputs := &persistence.FormInputs{
		PropertyPrice:         propertyPrice,
		DownPayment:           downPayment,
		InterestRate:          interestRate,
		DurationYears:         durationYears,
		InsuranceRate:         insuranceRate,
		NotaryRate:            notaryRate,
		AgencyRate:            agencyRate,
		AgencyFixed:           agencyFixed,
		BankFees:              bankFees,
		GuaranteeFees:         guaranteeFees,
		StartYear:             startYear,
		StartMonth:            startMonth,
		NetIncome1:            netIncome1,
		NetIncome2:            netIncome2,
		MonthlyRent:           monthlyRent,
		RentIncreaseRate:      rentIncreaseRate,
		SavingsRate:           savingsRate,
		InflationRate:         inflationRate,
		PropertyTax:           propertyTax,
		CondoFees:             condoFees,
		MaintenanceRate:       maintenanceRate,
		RenovationCost:        renovationCost,
		RenovationValueRate:   renovationValueRate,
		DownPayment1:          downPayment1,
		DownPayment2:          downPayment2,
		PaymentSplitMode:      paymentSplitMode,
		CurrentSalePrice:      currentSalePrice,
		CurrentLoanBalance:    currentLoanBalance,
		CurrentLoanLines:      currentLoanLinesJSON,
		CurrentOriginalLoan:    currentOriginalLoan,
		CurrentDownPayment1:    currentDownPayment1,
		CurrentRenovationCost:  currentRenovationCost,
		CurrentRenovationShare2: currentRenovationShare2,
		EarlyRepaymentPenalty:   earlyRepaymentPenalty,
		SalePropertyShare1:      salePropertyShare1,
		VirtualContribution2:    virtualContribution2,
		VirtualProfitShare2:     virtualProfitShare2,
		VirtualMonthlyPayment2:  virtualMonthlyPayment2,
		RFRYear2_1:              rfrYear2_1,
		RFRYear1_1:              rfrYear1_1,
		RFRYear2_2:              rfrYear2_2,
		RFRYear1_2:              rfrYear1_2,
		HouseholdSize:           householdSize,
		PropertyZone:            propertyZone,
		NewLoanLines:            newLoanLinesJSON,
		WorkLines:               workLinesJSON,
	}
	if err := h.store.Save(formInputs); err != nil {
		log.Printf("Failed to save inputs: %v", err)
	}

	input := model.CreditInput{
		PropertyPrice:         propertyPrice,
		LoanAmount:            loanAmount,
		InterestRate:          interestRate,
		DurationMonths:        durationYears * 12,
		InsuranceRate:         insuranceRate,
		NotaryRate:            notaryRate,
		AgencyRate:            agencyRate,
		AgencyFixed:           agencyFixed,
		BankFees:              bankFees,
		GuaranteeFees:         guaranteeFees,
		StartYear:             startYear,
		StartMonth:            startMonth,
		NetIncome1:            netIncome1,
		NetIncome2:            netIncome2,
		MonthlyRent:           monthlyRent,
		RentIncreaseRate:      rentIncreaseRate,
		SavingsRate:           savingsRate,
		InflationRate:         inflationRate,
		PropertyTax:           propertyTax,
		CondoFees:             condoFees,
		MaintenanceRate:       maintenanceRate,
		RenovationCost:        renovationCost,
		RenovationValueRate:   renovationValueRate,
		WorkLines:             workLines,
		DownPayment1:          downPayment1,
		DownPayment2:          downPayment2,
		PaymentSplitMode:      paymentSplitMode,
		CurrentSalePrice:      currentSalePrice,
		CurrentLoanBalance:    currentLoanBalance,
		CurrentLoanLines:      loanLines,
		EarlyRepaymentPenalty:  earlyRepaymentPenalty,
		CurrentDownPayment1:    currentDownPayment1,
		SalePropertyShare1:     salePropertyShare1,
		VirtualContribution2:   virtualContribution2,
		VirtualProfitShare2:    virtualProfitShare2,
		VirtualMonthlyPayment2: virtualMonthlyPayment2,
		RFRYear2_1:             rfrYear2_1,
		RFRYear1_1:             rfrYear1_1,
		RFRYear2_2:             rfrYear2_2,
		RFRYear1_2:             rfrYear1_2,
		HouseholdSize:          householdSize,
		PropertyZone:           propertyZone,
		NewLoanLines:           newLoanLines,
	}

	result := calculator.Calculate(input)

	data := struct {
		Input  model.CreditInput
		Result model.CreditResult
	}{
		Input:  input,
		Result: result,
	}

	if err := h.templates.ExecuteTemplate(w, "results.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "amortization.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "charts-unified.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "salecash.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "currentpropertychart.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseFloat(s string, fallback float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func parseInt(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
