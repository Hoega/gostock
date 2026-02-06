package handler

import (
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
	downPayment := parseFloat(r.FormValue("down_payment"), 0)
	loanAmount := propertyPrice - downPayment
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
	startYear := parseInt(r.FormValue("start_year"), 0)
	startMonth := parseInt(r.FormValue("start_month"), 0)
	netIncome1 := parseFloat(r.FormValue("net_income_1"), 0)
	netIncome2 := parseFloat(r.FormValue("net_income_2"), 0)
	monthlyRent := parseFloat(r.FormValue("monthly_rent"), 0)
	rentIncreaseRate := parseFloat(r.FormValue("rent_increase_rate"), 2.0)
	savingsRate := parseFloat(r.FormValue("savings_rate"), 0)
	propertyTax := parseFloat(r.FormValue("property_tax"), 0)
	condoFees := parseFloat(r.FormValue("condo_fees"), 0)
	renovationCost := parseFloat(r.FormValue("renovation_cost"), 0)
	renovationValueRate := parseFloat(r.FormValue("renovation_value_rate"), 70)

	// Save inputs to persistence
	formInputs := &persistence.FormInputs{
		PropertyPrice:    propertyPrice,
		DownPayment:      downPayment,
		InterestRate:     interestRate,
		DurationYears:    durationYears,
		InsuranceRate:    insuranceRate,
		NotaryRate:       notaryRate,
		AgencyRate:       agencyRate,
		AgencyFixed:      agencyFixed,
		BankFees:         bankFees,
		StartYear:        startYear,
		StartMonth:       startMonth,
		NetIncome1:       netIncome1,
		NetIncome2:       netIncome2,
		MonthlyRent:      monthlyRent,
		RentIncreaseRate: rentIncreaseRate,
		SavingsRate:      savingsRate,
		PropertyTax:      propertyTax,
		CondoFees:           condoFees,
		RenovationCost:      renovationCost,
		RenovationValueRate: renovationValueRate,
	}
	if err := h.store.Save(formInputs); err != nil {
		log.Printf("Failed to save inputs: %v", err)
	}

	input := model.CreditInput{
		PropertyPrice:    propertyPrice,
		LoanAmount:       loanAmount,
		InterestRate:     interestRate,
		DurationMonths:   durationYears * 12,
		InsuranceRate:    insuranceRate,
		NotaryRate:       notaryRate,
		AgencyRate:       agencyRate,
		AgencyFixed:      agencyFixed,
		BankFees:         bankFees,
		StartYear:        startYear,
		StartMonth:       startMonth,
		NetIncome1:       netIncome1,
		NetIncome2:       netIncome2,
		MonthlyRent:      monthlyRent,
		RentIncreaseRate: rentIncreaseRate,
		SavingsRate:      savingsRate,
		PropertyTax:      propertyTax,
		CondoFees:           condoFees,
		RenovationCost:      renovationCost,
		RenovationValueRate: renovationValueRate,
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

	if err := h.templates.ExecuteTemplate(w, "chart.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "rentvsbuy.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "salecash.html", data); err != nil {
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
