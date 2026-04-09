package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
)

type CompareHandler struct {
	templates *template.Template
	store     persistence.Store
}

func NewCompareHandler(templates *template.Template, store persistence.Store) *CompareHandler {
	return &CompareHandler{templates: templates, store: store}
}

// ShowForm renders the loan comparison page.
func (h *CompareHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	inputs, err := h.store.LoadCompareInputs()
	if err != nil {
		log.Printf("Failed to load compare inputs: %v", err)
		inputs = persistence.DefaultCompareInputs()
	}
	if err := h.templates.ExecuteTemplate(w, "compare.html", inputs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Calculate processes both offers and returns comparison results.
func (h *CompareHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	// Shared fields
	propertyPrice := cmpParseFloat(r.FormValue("property_price"), 250000)
	notaryRate := cmpParseFloat(r.FormValue("notary_rate"), 7.5)
	notaryFixed := cmpParseFloat(r.FormValue("notary_fixed"), 0)
	if notaryFixed > 0 && propertyPrice > 0 {
		notaryRate = notaryFixed / propertyPrice * 100
	}
	agencyRate := cmpParseFloat(r.FormValue("agency_rate"), 5.0)
	agencyFixed := cmpParseFloat(r.FormValue("agency_fixed"), 0)
	downPayment1 := cmpParseFloat(r.FormValue("down_payment_1"), 0)
	downPayment2 := cmpParseFloat(r.FormValue("down_payment_2"), 0)
	netIncome1 := cmpParseFloat(r.FormValue("net_income_1"), 0)
	netIncome2 := cmpParseFloat(r.FormValue("net_income_2"), 0)
	renovationCost := cmpParseFloat(r.FormValue("renovation_cost"), 0)
	workLinesJSON := r.FormValue("work_lines")
	var workLines []model.WorkLine
	if workLinesJSON != "" && workLinesJSON != "[]" {
		if err := json.Unmarshal([]byte(workLinesJSON), &workLines); err != nil {
			log.Printf("Failed to parse work lines JSON: %v", err)
		}
	}

	downPayment := downPayment1 + downPayment2
	notaryFees := propertyPrice * notaryRate / 100
	var agencyFees float64
	if agencyFixed > 0 {
		agencyFees = agencyFixed
	} else {
		agencyFees = propertyPrice * agencyRate / 100
	}
	baseLoanAmount := propertyPrice + notaryFees + agencyFees - downPayment

	// Parse offer A
	inputA := buildOfferInput(r, "a_", propertyPrice, baseLoanAmount, notaryRate, agencyRate, agencyFixed, downPayment1, downPayment2, netIncome1, netIncome2, renovationCost, workLines)
	// Parse offer B
	inputB := buildOfferInput(r, "b_", propertyPrice, baseLoanAmount, notaryRate, agencyRate, agencyFixed, downPayment1, downPayment2, netIncome1, netIncome2, renovationCost, workLines)

	resultA := calculator.Calculate(inputA)
	resultB := calculator.Calculate(inputB)

	// Save inputs
	formInputs := &persistence.CompareInputs{
		PropertyPrice:  propertyPrice,
		NotaryRate:     cmpParseFloat(r.FormValue("notary_rate"), 7.5),
		NotaryFixed:    notaryFixed,
		AgencyRate:     agencyRate,
		AgencyFixed:    agencyFixed,
		DownPayment1:   downPayment1,
		DownPayment2:   downPayment2,
		NetIncome1:     netIncome1,
		NetIncome2:     netIncome2,
		RenovationCost: renovationCost,
		WorkLines:      workLinesJSON,
		// Offer A
		InterestRateA:        inputA.InterestRate,
		DurationYearsA:       inputA.DurationMonths / 12,
		InsuranceRateA:       inputA.InsuranceRate,
		BankFeesA:            inputA.BankFees,
		GuaranteeFeesA:       inputA.GuaranteeFees,
		BrokerFeesA:          inputA.BrokerFees,
		StartYearA:           inputA.StartYear,
		StartMonthA:          inputA.StartMonth,
		NewLoanLinesA:        r.FormValue("a_new_loan_lines"),
		BridgeLoanEnabledA:     inputA.BridgeLoanEnabled,
		BridgeLoanSalePriceA:   inputA.CurrentSalePrice,
		BridgeLoanLoanBalanceA: inputA.CurrentLoanBalance,
		BridgeLoanQuotityA:     inputA.BridgeLoanQuotity,
		BridgeLoanRateA:        inputA.BridgeLoanRate,
		BridgeLoanDurationA:    cmpParseInt(r.FormValue("a_bridge_loan_duration"), 12),
		BridgeLoanInsuranceA:   inputA.BridgeLoanInsurance,
		BridgeLoanFranchiseA:   inputA.BridgeLoanFranchise,
		BridgeLoanSaleMonthA:   cmpParseInt(r.FormValue("a_bridge_loan_sale_month"), 12),
		BridgeLoanRepayPctA:    cmpParseFloat(r.FormValue("a_bridge_loan_repay_pct"), 100),
		BridgeLoanRepayLineA:   cmpParseInt(r.FormValue("a_bridge_loan_repay_line"), 0),
		// Offer B
		InterestRateB:        inputB.InterestRate,
		DurationYearsB:       inputB.DurationMonths / 12,
		InsuranceRateB:       inputB.InsuranceRate,
		BankFeesB:            inputB.BankFees,
		GuaranteeFeesB:       inputB.GuaranteeFees,
		BrokerFeesB:          inputB.BrokerFees,
		StartYearB:           inputB.StartYear,
		StartMonthB:          inputB.StartMonth,
		NewLoanLinesB:        r.FormValue("b_new_loan_lines"),
		BridgeLoanEnabledB:     inputB.BridgeLoanEnabled,
		BridgeLoanSalePriceB:   inputB.CurrentSalePrice,
		BridgeLoanLoanBalanceB: inputB.CurrentLoanBalance,
		BridgeLoanQuotityB:     inputB.BridgeLoanQuotity,
		BridgeLoanRateB:        inputB.BridgeLoanRate,
		BridgeLoanDurationB:    cmpParseInt(r.FormValue("b_bridge_loan_duration"), 12),
		BridgeLoanInsuranceB:   inputB.BridgeLoanInsurance,
		BridgeLoanFranchiseB:   inputB.BridgeLoanFranchise,
		BridgeLoanSaleMonthB:   cmpParseInt(r.FormValue("b_bridge_loan_sale_month"), 12),
		BridgeLoanRepayPctB:    cmpParseFloat(r.FormValue("b_bridge_loan_repay_pct"), 100),
		BridgeLoanRepayLineB:   cmpParseInt(r.FormValue("b_bridge_loan_repay_line"), 0),
	}
	if err := h.store.SaveCompareInputs(formInputs); err != nil {
		log.Printf("Failed to save compare inputs: %v", err)
	}

	// Compute bridge loan sale date hypotheses (3, 6, 12 months)
	// Shows state AFTER the sale: bridge repaid, surplus used for early repayment of main loan
	type BridgeHypothesis struct {
		Months        int
		CostA         model.BridgeLoanResult
		CostB         model.BridgeLoanResult
		MonthlyTotalA float64 // Mensualité totale après vente et remboursement anticipé
		MonthlyTotalB float64
	}
	var bridgeHypotheses []BridgeHypothesis
	if inputA.BridgeLoanEnabled || inputB.BridgeLoanEnabled {
		for _, m := range []int{3, 6, 12} {
			h := BridgeHypothesis{Months: m}
			if inputA.BridgeLoanEnabled && inputA.CurrentSalePrice > 0 {
				h.CostA = calculator.CalculateBridgeLoan(
					inputA.CurrentSalePrice, inputA.BridgeLoanQuotity,
					inputA.BridgeLoanRate, m, inputA.BridgeLoanInsurance,
					inputA.BridgeLoanFranchise, inputA.CurrentLoanBalance,
				)
				h.MonthlyTotalA = postSaleMonthlyTotal(inputA, resultA, m, h.CostA)
			}
			if inputB.BridgeLoanEnabled && inputB.CurrentSalePrice > 0 {
				h.CostB = calculator.CalculateBridgeLoan(
					inputB.CurrentSalePrice, inputB.BridgeLoanQuotity,
					inputB.BridgeLoanRate, m, inputB.BridgeLoanInsurance,
					inputB.BridgeLoanFranchise, inputB.CurrentLoanBalance,
				)
				h.MonthlyTotalB = postSaleMonthlyTotal(inputB, resultB, m, h.CostB)
			}
			bridgeHypotheses = append(bridgeHypotheses, h)
		}
	}

	data := struct {
		InputA           model.CreditInput
		InputB           model.CreditInput
		ResultA          model.CreditResult
		ResultB          model.CreditResult
		BridgeHypotheses []BridgeHypothesis
	}{
		InputA:           inputA,
		InputB:           inputB,
		ResultA:          resultA,
		ResultB:          resultB,
		BridgeHypotheses: bridgeHypotheses,
	}

	if err := h.templates.ExecuteTemplate(w, "compare-results.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "compare-chart.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// buildOfferInput builds a CreditInput from form values with the given prefix.
func buildOfferInput(r *http.Request, prefix string, propertyPrice, baseLoanAmount, notaryRate, agencyRate, agencyFixed, downPayment1, downPayment2, netIncome1, netIncome2, renovationCost float64, workLines []model.WorkLine) model.CreditInput {
	interestRate := cmpParseFloat(r.FormValue(prefix+"interest_rate"), 3.5)
	durationYears := cmpParseInt(r.FormValue(prefix+"duration_years"), 20)
	insuranceRate := cmpParseFloat(r.FormValue(prefix+"insurance_rate"), 0.34)
	bankFees := cmpParseFloat(r.FormValue(prefix+"bank_fees"), 0)
	guaranteeFees := cmpParseFloat(r.FormValue(prefix+"guarantee_fees"), 0)
	brokerFees := cmpParseFloat(r.FormValue(prefix+"broker_fees"), 0)
	startYear := cmpParseInt(r.FormValue(prefix+"start_year"), 0)
	startMonth := cmpParseInt(r.FormValue(prefix+"start_month"), 0)

	// Parse loan lines
	newLoanLinesJSON := r.FormValue(prefix + "new_loan_lines")
	var newLoanLines []model.NewLoanLine
	if newLoanLinesJSON != "" && newLoanLinesJSON != "[]" {
		if err := json.Unmarshal([]byte(newLoanLinesJSON), &newLoanLines); err != nil {
			log.Printf("Failed to parse %s loan lines JSON: %v", prefix, err)
		}
		// Compute proper tiers for lines that have zero-payment tiers (from frontend default)
		for i := range newLoanLines {
			line := &newLoanLines[i]
			if line.Amount > 0 && line.DurationYears > 0 && needsComputedTiers(line.Tiers) {
				dm := line.DurationYears * 12
				mr := line.Rate / 100 / 12
				var mp float64
				if mr == 0 {
					mp = line.Amount / float64(dm)
				} else {
					mp = line.Amount * mr / (1 - math.Pow(1+mr, float64(-dm)))
				}
				line.Tiers = []model.PaymentTier{{StartMonth: 1, EndMonth: dm, MonthlyPayment: math.Round(mp*100) / 100}}
			}
		}
	}

	// Parse bridge loan
	bridgeLoanEnabled := r.FormValue(prefix+"bridge_loan_enabled") == "on" || r.FormValue(prefix+"bridge_loan_enabled") == "true"
	bridgeLoanSalePrice := cmpParseFloat(r.FormValue(prefix+"bridge_loan_sale_price"), 0)
	bridgeLoanLoanBalance := cmpParseFloat(r.FormValue(prefix+"bridge_loan_loan_balance"), 0)
	bridgeLoanQuotity := cmpParseFloat(r.FormValue(prefix+"bridge_loan_quotity"), 70)
	bridgeLoanRate := cmpParseFloat(r.FormValue(prefix+"bridge_loan_rate"), 3.5)
	bridgeLoanDuration := cmpParseInt(r.FormValue(prefix+"bridge_loan_duration"), 12)
	bridgeLoanInsurance := cmpParseFloat(r.FormValue(prefix+"bridge_loan_insurance"), 0.34)
	bridgeLoanFranchise := r.FormValue(prefix + "bridge_loan_franchise")
	if bridgeLoanFranchise == "" {
		bridgeLoanFranchise = "partielle"
	}
	bridgeLoanSaleMonth := cmpParseInt(r.FormValue(prefix+"bridge_loan_sale_month"), bridgeLoanDuration)
	bridgeLoanRepayPct := cmpParseFloat(r.FormValue(prefix+"bridge_loan_repay_pct"), 100)
	bridgeLoanRepayLine := cmpParseInt(r.FormValue(prefix+"bridge_loan_repay_line"), 0)

	// Compute final loan amount: base + finançable fees - bridge net contribution
	// The bridge finances both the rachat and part of the acquisition
	loanAmount := baseLoanAmount + guaranteeFees + bankFees
	if bridgeLoanEnabled && bridgeLoanLoanBalance > 0 {
		bridgeAmount := bridgeLoanSalePrice * bridgeLoanQuotity / 100
		bridgeNetContribution := bridgeAmount - bridgeLoanLoanBalance
		if bridgeNetContribution > 0 {
			loanAmount -= bridgeNetContribution
		}
	}
	if loanAmount < 0 {
		loanAmount = 0
	}

	// Auto-prepend "Prêt principal" line if loan lines don't cover the full amount
	if len(newLoanLines) > 0 {
		var sumLines float64
		for _, line := range newLoanLines {
			sumLines += line.Amount
		}
		mainAmount := loanAmount - sumLines
		if mainAmount > 0 {
			// Compute the actual monthly payment for the main loan line
			durationMonths := durationYears * 12
			monthlyRate := interestRate / 100 / 12
			var monthlyPayment float64
			if monthlyRate == 0 {
				monthlyPayment = mainAmount / float64(durationMonths)
			} else {
				monthlyPayment = mainAmount * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-durationMonths)))
			}
			mainLine := model.NewLoanLine{
				Label:         "Prêt principal",
				Amount:        mainAmount,
				Rate:          interestRate,
				DurationYears: durationYears,
				InsuranceRate: insuranceRate,
				Tiers:         []model.PaymentTier{{StartMonth: 1, EndMonth: durationMonths, MonthlyPayment: math.Round(monthlyPayment*100) / 100}},
			}
			newLoanLines = append([]model.NewLoanLine{mainLine}, newLoanLines...)
		}
	}

	// Clamp bridge sale month for cost calculation
	if bridgeLoanEnabled && bridgeLoanSalePrice > 0 {
		if bridgeLoanSaleMonth < 1 {
			bridgeLoanSaleMonth = 1
		}
		if bridgeLoanSaleMonth > bridgeLoanDuration {
			bridgeLoanSaleMonth = bridgeLoanDuration
		}
		bridgeLoanDuration = bridgeLoanSaleMonth
	}

	return model.CreditInput{
		PropertyPrice:       propertyPrice,
		LoanAmount:          loanAmount,
		InterestRate:        interestRate,
		DurationMonths:      durationYears * 12,
		InsuranceRate:       insuranceRate,
		NotaryRate:          notaryRate,
		AgencyRate:          agencyRate,
		DownPayment1:        downPayment1,
		DownPayment2:        downPayment2,
		AgencyFixed:         agencyFixed,
		BankFees:            bankFees,
		GuaranteeFees:       guaranteeFees,
		BrokerFees:          brokerFees,
		StartYear:           startYear,
		StartMonth:          startMonth,
		NetIncome1:          netIncome1,
		NetIncome2:          netIncome2,
		RenovationCost:      renovationCost,
		WorkLines:           workLines,
		NewLoanLines:        newLoanLines,
		CurrentSalePrice:    bridgeLoanSalePrice,
		CurrentLoanBalance:  bridgeLoanLoanBalance,
		BridgeLoanEnabled:   bridgeLoanEnabled,
		BridgeLoanQuotity:   bridgeLoanQuotity,
		BridgeLoanRate:      bridgeLoanRate,
		BridgeLoanDuration:  bridgeLoanDuration,
		BridgeLoanInsurance: bridgeLoanInsurance,
		BridgeLoanFranchise: bridgeLoanFranchise,
		BridgeLoanSaleMonth: bridgeLoanSaleMonth,
		BridgeLoanRepayPct:  bridgeLoanRepayPct,
		BridgeLoanRepayLine: bridgeLoanRepayLine,
	}
}

// needsComputedTiers returns true if tiers are empty or all have zero payment.
func needsComputedTiers(tiers []model.PaymentTier) bool {
	if len(tiers) == 0 {
		return true
	}
	for _, t := range tiers {
		if t.MonthlyPayment > 0 {
			return false
		}
	}
	return true
}

// postSaleMonthlyTotal computes the new monthly total after selling the old property
// at the given month: bridge is repaid, surplus goes to early repayment of the main loan.
func postSaleMonthlyTotal(input model.CreditInput, result model.CreditResult, saleMonth int, bridge model.BridgeLoanResult) float64 {
	// Surplus from sale after repaying the bridge loan
	surplus := input.CurrentSalePrice - bridge.CapitalizedAmount
	if surplus <= 0 {
		return result.MonthlyTotal
	}
	earlyRepayment := surplus * input.BridgeLoanRepayPct / 100

	// Get remaining balance on the main loan at sale month
	if saleMonth < 1 || saleMonth > len(result.Amortization) {
		return result.MonthlyTotal
	}
	remainingBalance := result.Amortization[saleMonth-1].RemainingBalance
	newBalance := remainingBalance - earlyRepayment
	if newBalance <= 0 {
		return 0
	}

	// Recalculate monthly payment for remaining months
	remainingMonths := input.DurationMonths - saleMonth
	if remainingMonths <= 0 {
		return 0
	}
	monthlyRate := input.InterestRate / 100 / 12
	var newPayment float64
	if monthlyRate == 0 {
		newPayment = newBalance / float64(remainingMonths)
	} else {
		newPayment = newBalance * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-remainingMonths)))
	}

	// Insurance stays on initial capital (French convention)
	monthlyInsurance := input.InsuranceRate / 100 / 12 * input.LoanAmount

	return math.Round((newPayment+monthlyInsurance)*100) / 100
}

func cmpParseFloat(s string, fallback float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func cmpParseInt(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
