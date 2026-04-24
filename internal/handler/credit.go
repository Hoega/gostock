package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	texttemplate "text/template"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
)

type CreditHandler struct {
	templates   *template.Template
	mdTemplates *texttemplate.Template
	store       persistence.Store
}

func NewCreditHandler(templates *template.Template, mdTemplates *texttemplate.Template, store persistence.Store) *CreditHandler {
	return &CreditHandler{templates: templates, mdTemplates: mdTemplates, store: store}
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
	input, ok := h.buildInputFromForm(w, r)
	if !ok {
		return
	}
	h.renderResultPartials(w, input)
}

// buildInputFromForm parses the form, saves to DB, and returns a CreditInput ready for Calculate.
// On error, writes HTTP error and returns ok=false.
func (h *CreditHandler) buildInputFromForm(w http.ResponseWriter, r *http.Request) (model.CreditInput, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return model.CreditInput{}, false
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

	currentLoanStartYear := parseInt(r.FormValue("current_loan_start_year"), 2020)
	currentLoanStartMonth := parseInt(r.FormValue("current_loan_start_month"), 1)

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
	virtualPaymentTiers2JSON := r.FormValue("virtual_payment_tiers_2")
	var virtualPaymentTiers2 []model.PaymentTier
	if virtualPaymentTiers2JSON != "" && virtualPaymentTiers2JSON != "[]" {
		if err := json.Unmarshal([]byte(virtualPaymentTiers2JSON), &virtualPaymentTiers2); err != nil {
			log.Printf("Failed to parse virtual payment tiers 2 JSON: %v", err)
			virtualPaymentTiers2 = []model.PaymentTier{}
		}
	}

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
	brokerFees := parseFloat(r.FormValue("broker_fees"), 0)
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

	// Parse energy comparison fields
	energy1Gas := parseFloat(r.FormValue("energy_1_gas"), 0)
	energy1Electricity := parseFloat(r.FormValue("energy_1_electricity"), 0)
	energy1GasKWh := parseFloat(r.FormValue("energy_1_gas_kwh"), 0)
	energy1ElectricityKWh := parseFloat(r.FormValue("energy_1_electricity_kwh"), 0)
	energy1Other := parseFloat(r.FormValue("energy_1_other"), 0)
	energy1OtherLabel := r.FormValue("energy_1_other_label")
	energy1Label := r.FormValue("energy_1_label")
	if energy1Label == "" {
		energy1Label = "Bien actuel"
	}
	energy2Gas := parseFloat(r.FormValue("energy_2_gas"), 0)
	energy2Electricity := parseFloat(r.FormValue("energy_2_electricity"), 0)
	energy2GasKWh := parseFloat(r.FormValue("energy_2_gas_kwh"), 0)
	energy2ElectricityKWh := parseFloat(r.FormValue("energy_2_electricity_kwh"), 0)
	energy2Other := parseFloat(r.FormValue("energy_2_other"), 0)
	energy2OtherLabel := r.FormValue("energy_2_other_label")
	energy2Label := r.FormValue("energy_2_label")
	if energy2Label == "" {
		energy2Label = "Nouveau bien"
	}
	energy1Surface := parseFloat(r.FormValue("energy_1_surface"), 0)
	energy1DPE := parseFloat(r.FormValue("energy_1_dpe"), 0)
	energy2Surface := parseFloat(r.FormValue("energy_2_surface"), 0)
	energy2DPE := parseFloat(r.FormValue("energy_2_dpe"), 0)
	energy3Gas := parseFloat(r.FormValue("energy_3_gas"), 0)
	energy3Electricity := parseFloat(r.FormValue("energy_3_electricity"), 0)
	energy3GasKWh := parseFloat(r.FormValue("energy_3_gas_kwh"), 0)
	energy3ElectricityKWh := parseFloat(r.FormValue("energy_3_electricity_kwh"), 0)
	energy3Other := parseFloat(r.FormValue("energy_3_other"), 0)
	energy3OtherLabel := r.FormValue("energy_3_other_label")
	energy3Label := r.FormValue("energy_3_label")
	if energy3Label == "" {
		energy3Label = "Bien 3"
	}
	energy3Surface := parseFloat(r.FormValue("energy_3_surface"), 0)
	energy3DPE := parseFloat(r.FormValue("energy_3_dpe"), 0)
	energyPriceIncrease := parseFloat(r.FormValue("energy_price_increase"), 4.0)

	// Parse bridge loan fields
	bridgeLoanEnabled := r.FormValue("bridge_loan_enabled") == "on" || r.FormValue("bridge_loan_enabled") == "true"
	bridgeLoanQuotity := parseFloat(r.FormValue("bridge_loan_quotity"), 70)
	bridgeLoanRate := parseFloat(r.FormValue("bridge_loan_rate"), 3.5)
	bridgeLoanDuration := parseInt(r.FormValue("bridge_loan_duration"), 12)
	bridgeLoanInsurance := parseFloat(r.FormValue("bridge_loan_insurance"), 0.34)
	bridgeLoanFranchise := r.FormValue("bridge_loan_franchise")
	if bridgeLoanFranchise == "" {
		bridgeLoanFranchise = "partielle"
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
		BrokerFees:            brokerFees,
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
		CurrentSalePrice:       currentSalePrice,
		CurrentLoanBalance:     currentLoanBalance,
		CurrentLoanLines:       currentLoanLinesJSON,
		CurrentLoanStartYear:   currentLoanStartYear,
		CurrentLoanStartMonth:  currentLoanStartMonth,
		CurrentOriginalLoan:    currentOriginalLoan,
		CurrentDownPayment1:    currentDownPayment1,
		CurrentRenovationCost:  currentRenovationCost,
		CurrentRenovationShare2: currentRenovationShare2,
		EarlyRepaymentPenalty:   earlyRepaymentPenalty,
		SalePropertyShare1:      salePropertyShare1,
		VirtualContribution2:    virtualContribution2,
		VirtualProfitShare2:     virtualProfitShare2,
		VirtualMonthlyPayment2:  virtualMonthlyPayment2,
		VirtualPaymentTiers2:    virtualPaymentTiers2JSON,
		RFRYear2_1:              rfrYear2_1,
		RFRYear1_1:              rfrYear1_1,
		RFRYear2_2:              rfrYear2_2,
		RFRYear1_2:              rfrYear1_2,
		HouseholdSize:           householdSize,
		PropertyZone:            propertyZone,
		NewLoanLines:            newLoanLinesJSON,
		WorkLines:               workLinesJSON,
		Energy1Gas:              energy1Gas,
		Energy1Electricity:      energy1Electricity,
		Energy1GasKWh:           energy1GasKWh,
		Energy1ElectricityKWh:   energy1ElectricityKWh,
		Energy1Other:            energy1Other,
		Energy1OtherLabel:       energy1OtherLabel,
		Energy1Label:            energy1Label,
		Energy2Gas:              energy2Gas,
		Energy2Electricity:      energy2Electricity,
		Energy2GasKWh:           energy2GasKWh,
		Energy2ElectricityKWh:   energy2ElectricityKWh,
		Energy2Other:            energy2Other,
		Energy2OtherLabel:       energy2OtherLabel,
		Energy2Label:            energy2Label,
		Energy1Surface:          energy1Surface,
		Energy1DPE:              energy1DPE,
		Energy2Surface:          energy2Surface,
		Energy2DPE:              energy2DPE,
		Energy3Gas:              energy3Gas,
		Energy3Electricity:      energy3Electricity,
		Energy3GasKWh:           energy3GasKWh,
		Energy3ElectricityKWh:   energy3ElectricityKWh,
		Energy3Other:            energy3Other,
		Energy3OtherLabel:       energy3OtherLabel,
		Energy3Label:            energy3Label,
		Energy3Surface:          energy3Surface,
		Energy3DPE:              energy3DPE,
		EnergyPriceIncrease:     energyPriceIncrease,
		BridgeLoanEnabled:       bridgeLoanEnabled,
		BridgeLoanQuotity:       bridgeLoanQuotity,
		BridgeLoanRate:           bridgeLoanRate,
		BridgeLoanDuration:      bridgeLoanDuration,
		BridgeLoanInsurance:     bridgeLoanInsurance,
		BridgeLoanFranchise:     bridgeLoanFranchise,
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
		BrokerFees:            brokerFees,
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
		CurrentSalePrice:       currentSalePrice,
		CurrentLoanBalance:     currentLoanBalance,
		CurrentLoanLines:       loanLines,
		CurrentLoanStartYear:   currentLoanStartYear,
		CurrentLoanStartMonth:  currentLoanStartMonth,
		EarlyRepaymentPenalty:  earlyRepaymentPenalty,
		CurrentDownPayment1:    currentDownPayment1,
		SalePropertyShare1:     salePropertyShare1,
		VirtualContribution2:   virtualContribution2,
		VirtualProfitShare2:    virtualProfitShare2,
		VirtualMonthlyPayment2: virtualMonthlyPayment2,
		VirtualPaymentTiers2:   virtualPaymentTiers2,
		RFRYear2_1:             rfrYear2_1,
		RFRYear1_1:             rfrYear1_1,
		RFRYear2_2:             rfrYear2_2,
		RFRYear1_2:             rfrYear1_2,
		HouseholdSize:          householdSize,
		PropertyZone:           propertyZone,
		NewLoanLines:           newLoanLines,
		Energy1Gas:             energy1Gas,
		Energy1Electricity:     energy1Electricity,
		Energy1GasKWh:          energy1GasKWh,
		Energy1ElectricityKWh:  energy1ElectricityKWh,
		Energy1Other:           energy1Other,
		Energy1OtherLabel:      energy1OtherLabel,
		Energy1Label:           energy1Label,
		Energy2Gas:             energy2Gas,
		Energy2Electricity:     energy2Electricity,
		Energy2GasKWh:          energy2GasKWh,
		Energy2ElectricityKWh:  energy2ElectricityKWh,
		Energy2Other:           energy2Other,
		Energy2OtherLabel:      energy2OtherLabel,
		Energy2Label:           energy2Label,
		Energy1Surface:         energy1Surface,
		Energy1DPE:             energy1DPE,
		Energy2Surface:         energy2Surface,
		Energy2DPE:             energy2DPE,
		Energy3Gas:             energy3Gas,
		Energy3Electricity:     energy3Electricity,
		Energy3GasKWh:          energy3GasKWh,
		Energy3ElectricityKWh:  energy3ElectricityKWh,
		Energy3Other:           energy3Other,
		Energy3OtherLabel:      energy3OtherLabel,
		Energy3Label:           energy3Label,
		Energy3Surface:         energy3Surface,
		Energy3DPE:             energy3DPE,
		EnergyPriceIncrease:    energyPriceIncrease,
		BridgeLoanEnabled:      bridgeLoanEnabled,
		BridgeLoanQuotity:      bridgeLoanQuotity,
		BridgeLoanRate:         bridgeLoanRate,
		BridgeLoanDuration:     bridgeLoanDuration,
		BridgeLoanInsurance:    bridgeLoanInsurance,
		BridgeLoanFranchise:    bridgeLoanFranchise,
	}

	return input, true
}

// renderResultPartials runs the calculator and writes all HTMX partial templates.
func (h *CreditHandler) renderResultPartials(w http.ResponseWriter, input model.CreditInput) {
	result := calculator.Calculate(input)

	data := struct {
		Input  model.CreditInput
		Result model.CreditResult
	}{
		Input:  input,
		Result: result,
	}

	partials := []string{
		"results.html",
		"amortization.html",
		"charts-unified.html",
		"loancomposition.html",
		"paymentbreakdown.html",
		"salecash.html",
		"downpayment-impact.html",
		"energy-comparison.html",
	}
	for _, name := range partials {
		if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// ExportMarkdown parses the form, runs the calculator, and streams a Markdown file.
func (h *CreditHandler) ExportMarkdown(w http.ResponseWriter, r *http.Request) {
	input, ok := h.buildInputFromForm(w, r)
	if !ok {
		return
	}
	result := calculator.Calculate(input)
	data := struct {
		Input  model.CreditInput
		Result model.CreditResult
	}{Input: input, Result: result}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="simulation-credit.md"`)
	if err := h.mdTemplates.ExecuteTemplate(w, "credit-export.md", data); err != nil {
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
