package calculator

import (
	"fmt"
	"math"
	"time"

	"github.com/Hoega/gostock/internal/model"
)

var frenchMonths = [12]string{
	"Janvier", "Février", "Mars", "Avril", "Mai", "Juin",
	"Juillet", "Août", "Septembre", "Octobre", "Novembre", "Décembre",
}

// PresetWorkCategories contains the predefined work categories with their valuation parameters.
// Each category has an initial valuation rate and an annual appreciation/depreciation rate.
var PresetWorkCategories = []model.WorkCategory{
	{ID: "structure", Label: "Gros œuvre / Structure", InitialRate: 80, AnnualRate: 0.5},
	{ID: "extension", Label: "Extension / Agrandissement", InitialRate: 100, AnnualRate: 1.0},
	{ID: "isolation", Label: "Isolation / Énergie", InitialRate: 90, AnnualRate: 1.5},
	{ID: "cuisine", Label: "Cuisine équipée", InitialRate: 100, AnnualRate: -2.0},
	{ID: "sdb", Label: "Salle de bain", InitialRate: 90, AnnualRate: -2.0},
	{ID: "peinture", Label: "Peinture / Décoration", InitialRate: 50, AnnualRate: -10.0},
	{ID: "exterieur", Label: "Extérieur (jardin, terrasse)", InitialRate: 60, AnnualRate: -3.0},
	{ID: "autres", Label: "Autres travaux", InitialRate: 70, AnnualRate: 0.0},
	{ID: "legacy", Label: "Travaux (mode simple)", InitialRate: 100, AnnualRate: 0.0}, // For backward compatibility
}

// GetWorkCategory returns the work category by ID, or the default "autres" category if not found.
func GetWorkCategory(id string) model.WorkCategory {
	for _, cat := range PresetWorkCategories {
		if cat.ID == id {
			return cat
		}
	}
	// Default to "autres" category
	return PresetWorkCategories[len(PresetWorkCategories)-1]
}

// CalculateWorkValueAtYear computes the value of a work line at a given year.
// Formula: Amount × InitialRate × (1 + AnnualRate)^year with floor at 0.
func CalculateWorkValueAtYear(work model.WorkLine, year int) float64 {
	cat := GetWorkCategory(work.CategoryID)
	initialValue := work.Amount * cat.InitialRate / 100
	value := initialValue * math.Pow(1+cat.AnnualRate/100, float64(year))
	if value < 0 {
		return 0
	}
	return value
}

// CalculateTotalWorkValueAtYear computes the total value of all work lines at a given year.
func CalculateTotalWorkValueAtYear(workLines []model.WorkLine, year int) float64 {
	var total float64
	for _, work := range workLines {
		total += CalculateWorkValueAtYear(work, year)
	}
	return total
}

// ConvertLegacyToWorkLines converts legacy RenovationCost/RenovationValueRate to WorkLines.
// This provides backward compatibility with existing data.
func ConvertLegacyToWorkLines(renovationCost, renovationValueRate float64) []model.WorkLine {
	if renovationCost <= 0 {
		return nil
	}
	// Use "autres" category as base, but adjust initial rate to match the legacy value rate
	// For legacy compatibility, we create a single "autres" work line
	return []model.WorkLine{
		{
			CategoryID: "autres",
			Label:      "Travaux (mode simple)",
			Amount:     renovationCost,
		},
	}
}

// tierLoanState tracks the amortization state for a single loan with payment tiers.
type tierLoanState struct {
	label            string
	amount           float64
	rate             float64
	monthlyRate      float64
	durationMonths   int
	insuranceRate    float64
	balance          float64
	monthlyInsurance float64
	deferralMonths   int     // Nombre de mois de différé (paiement intérêts seuls)
	deferralRate     float64 // Taux mensuel pour les intérêts intercalaires
	tiers            []model.PaymentTier
}

// getPaymentForMonth returns the monthly payment for a given month based on tiers.
// Returns 0 if no tier covers the month.
func getPaymentForMonth(tiers []model.PaymentTier, month int) float64 {
	for _, tier := range tiers {
		if month >= tier.StartMonth && month <= tier.EndMonth {
			return tier.MonthlyPayment
		}
	}
	return 0
}

// calculateE2AccumulatedContribution calculates the total contribution of Borrower 2 up to a given month.
// If tiers is empty, uses flatMonthlyPayment * month for backward compatibility.
// Otherwise, sums the payments from each tier month by month.
func calculateE2AccumulatedContribution(initialContribution, flatMonthlyPayment float64, tiers []model.PaymentTier, month int) float64 {
	if len(tiers) == 0 {
		// Backward compatibility: use flat monthly payment
		return initialContribution + flatMonthlyPayment*float64(month)
	}

	// Sum payments from tiers month by month
	var totalPayments float64
	for m := 1; m <= month; m++ {
		totalPayments += getPaymentForMonth(tiers, m)
	}
	return initialContribution + totalPayments
}

// calculateTierBasedSchedule computes the month-by-month payment schedule using manual tiers.
// Each loan line has user-defined payment tiers specifying the payment for each period.
func calculateTierBasedSchedule(lines []model.NewLoanLine, defaultDurationMonths int) []model.MonthlySchedule {
	if len(lines) == 0 {
		return nil
	}

	// Initialize loan states
	states := make([]*tierLoanState, 0, len(lines))
	var maxDuration int

	for _, line := range lines {
		if line.Amount <= 0 {
			continue
		}
		durationMonths := line.DurationYears*12 + line.DeferralMonths
		if durationMonths <= 0 {
			durationMonths = defaultDurationMonths
		}
		if durationMonths > maxDuration {
			maxDuration = durationMonths
		}

		monthlyRate := line.Rate / 100 / 12
		monthlyInsurance := line.InsuranceRate / 100 / 12 * line.Amount

		// Taux d'intérêts intercalaires : utiliser DeferralRate si défini, sinon Rate
		deferralRate := line.DeferralRate
		if deferralRate == 0 {
			deferralRate = line.Rate
		}
		monthlyDeferralRate := deferralRate / 100 / 12

		state := &tierLoanState{
			label:            line.Label,
			amount:           line.Amount,
			rate:             line.Rate,
			monthlyRate:      monthlyRate,
			durationMonths:   durationMonths,
			insuranceRate:    line.InsuranceRate,
			balance:          line.Amount,
			monthlyInsurance: monthlyInsurance,
			deferralMonths:   line.DeferralMonths,
			deferralRate:     monthlyDeferralRate,
			tiers:            line.Tiers,
		}
		states = append(states, state)
	}

	if len(states) == 0 {
		return nil
	}

	schedule := make([]model.MonthlySchedule, 0, maxDuration)

	for month := 1; month <= maxDuration; month++ {
		payments := make([]model.LoanMonthPayment, 0, len(states))
		var monthTotal float64

		for _, s := range states {
			if month > s.durationMonths || s.balance <= 0 {
				payments = append(payments, model.LoanMonthPayment{
					Label: s.label,
				})
				continue
			}

			insurance := s.monthlyInsurance

			var principal float64
			var tierPayment float64
			var interest float64

			// Pendant le différé : paiement des intérêts intercalaires seuls (pas de capital)
			if month <= s.deferralMonths {
				interest = s.balance * s.deferralRate
				tierPayment = interest
				principal = 0
			} else {
				interest = s.balance * s.monthlyRate
				// Get payment from tier - offset by deferral months
				amortizationMonth := month - s.deferralMonths
				tierPayment = getPaymentForMonth(s.tiers, amortizationMonth)
				// Payment covers: principal + interest (insurance is separate)
				principal = tierPayment - interest
				if principal < 0 {
					principal = 0
				}
				if principal > s.balance {
					principal = s.balance
				}
			}

			total := tierPayment + insurance
			s.balance -= principal
			if s.balance < 0.01 {
				s.balance = 0
			}

			payments = append(payments, model.LoanMonthPayment{
				Label:     s.label,
				Principal: round2(principal),
				Interest:  round2(interest),
				Insurance: round2(insurance),
				Total:     round2(total),
			})
			monthTotal += total
		}

		schedule = append(schedule, model.MonthlySchedule{
			Month:       month,
			Payments:    payments,
			TotalAmount: round2(monthTotal),
		})
	}

	return schedule
}

// calculateLoanLineTotals computes totals for a loan line based on its tiers.
func calculateLoanLineTotals(amount float64, rate float64, durationMonths int, insuranceRate float64, deferralMonths int, deferralRate float64, tiers []model.PaymentTier) (totalInterest, totalInsurance float64) {
	monthlyRate := rate / 100 / 12
	monthlyInsurance := insuranceRate / 100 / 12 * amount

	// Taux d'intérêts intercalaires : utiliser deferralRate si défini, sinon rate
	if deferralRate == 0 {
		deferralRate = rate
	}
	monthlyDeferralRate := deferralRate / 100 / 12

	remaining := amount
	for m := 1; m <= durationMonths && remaining > 0; m++ {
		var interest float64
		var principal float64

		// Pendant le différé : intérêts intercalaires, pas de remboursement du capital
		if m <= deferralMonths {
			interest = remaining * monthlyDeferralRate
			principal = 0
		} else {
			interest = remaining * monthlyRate
			payment := getPaymentForMonth(tiers, m)
			principal = payment - interest
			if principal < 0 {
				principal = 0
			}
			if principal > remaining {
				principal = remaining
			}
		}
		remaining -= principal
		totalInterest += interest
	}

	totalInsurance = monthlyInsurance * float64(durationMonths)
	return
}

// Calculate computes the full mortgage simulation from the given input.
func Calculate(input model.CreditInput) model.CreditResult {
	var monthlyPayment, monthlyInsurance, totalInterest, totalInsurance float64
	var capital float64
	var n int
	var loanLineResults []model.NewLoanLineResult

	// Check if we have multiple loan lines
	if len(input.NewLoanLines) > 0 {
		// Calculate each loan line separately using tier-based approach
		for _, line := range input.NewLoanLines {
			if line.Amount <= 0 {
				continue
			}
			durationMonths := line.DurationYears*12 + line.DeferralMonths
			if durationMonths <= 0 {
				durationMonths = input.DurationMonths
			}

			// Calculate totals based on tiers
			ti, tis := calculateLoanLineTotals(line.Amount, line.Rate, durationMonths, line.InsuranceRate, line.DeferralMonths, line.DeferralRate, line.Tiers)

			// Calculate average monthly payment from tiers for display
			var totalPayments float64
			var activeMonths int
			for _, tier := range line.Tiers {
				months := tier.EndMonth - tier.StartMonth + 1
				if months > 0 {
					totalPayments += tier.MonthlyPayment * float64(months)
					activeMonths += months
				}
			}
			var avgMonthlyPayment float64
			if activeMonths > 0 {
				avgMonthlyPayment = totalPayments / float64(activeMonths)
			}

			mi := line.InsuranceRate / 100 / 12 * line.Amount

			loanLineResults = append(loanLineResults, model.NewLoanLineResult{
				Label:            line.Label,
				Amount:           line.Amount,
				Rate:             line.Rate,
				DurationYears:    line.DurationYears,
				DeferralMonths:   line.DeferralMonths,
				InsuranceRate:    line.InsuranceRate,
				MonthlyPayment:   round2(avgMonthlyPayment),
				MonthlyInsurance: round2(mi),
				MonthlyTotal:     round2(avgMonthlyPayment + mi),
				TotalInterest:    round2(ti),
				TotalInsurance:   round2(tis),
			})

			monthlyPayment += avgMonthlyPayment
			monthlyInsurance += mi
			totalInterest += ti
			totalInsurance += tis
			capital += line.Amount

			// Track the longest duration for amortization table
			if durationMonths > n {
				n = durationMonths
			}
		}
	} else {
		// Use single loan parameters (backward compatibility)
		// No tiers = assume constant payment
		capital = input.LoanAmount
		n = input.DurationMonths
		monthlyRate := input.InterestRate / 100 / 12
		if monthlyRate == 0 {
			monthlyPayment = capital / float64(n)
		} else {
			monthlyPayment = capital * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-n)))
		}
		monthlyInsurance = input.InsuranceRate / 100 / 12 * capital

		// Calculate total interest
		remaining := capital
		for m := 1; m <= n; m++ {
			interest := remaining * monthlyRate
			principal := monthlyPayment - interest
			remaining -= principal
			totalInterest += interest
		}
		totalInsurance = monthlyInsurance * float64(n)
	}

	// Ensure we have a duration
	if n == 0 {
		n = input.DurationMonths
	}

	// Notary fees
	notaryFees := input.PropertyPrice * input.NotaryRate / 100

	// Agency fees: fixed amount takes priority, otherwise percentage
	agencyFees := input.AgencyFixed
	if agencyFees == 0 {
		agencyFees = input.PropertyPrice * input.AgencyRate / 100
	}

	// Determine start date
	startYear := input.StartYear
	startMonth := input.StartMonth
	if startYear == 0 {
		now := time.Now()
		startYear = now.Year()
		startMonth = int(now.Month())
	}

	// Build amortization table
	// For multiple loans, we use a simplified table with the primary loan rate
	// or an average rate weighted by amount
	var avgMonthlyRate float64
	if len(input.NewLoanLines) > 0 && capital > 0 {
		var weightedRate float64
		for _, line := range input.NewLoanLines {
			if line.Amount > 0 {
				weightedRate += line.Rate * line.Amount
			}
		}
		avgMonthlyRate = (weightedRate / capital) / 100 / 12
	} else {
		avgMonthlyRate = input.InterestRate / 100 / 12
	}

	remaining := capital
	amortization := make([]model.AmortizationRow, 0, n)

	// Calculate monthly payment for amortization display (using average rate)
	var amortMonthlyPayment float64
	if avgMonthlyRate == 0 {
		amortMonthlyPayment = capital / float64(n)
	} else {
		amortMonthlyPayment = capital * avgMonthlyRate / (1 - math.Pow(1+avgMonthlyRate, float64(-n)))
	}

	for m := 1; m <= n; m++ {
		// Compute date for this row
		month := (startMonth-1+m-1)%12 + 1
		year := startYear + (startMonth-1+m-1)/12
		date := fmt.Sprintf("%s %d", frenchMonths[month-1], year)

		interest := remaining * avgMonthlyRate
		principal := amortMonthlyPayment - interest
		remaining -= principal

		// Avoid floating point noise on the last month
		if m == n {
			principal += remaining
			remaining = 0
		}

		amortization = append(amortization, model.AmortizationRow{
			Month:            m,
			Date:             date,
			Payment:          round2(monthlyPayment),
			Principal:        round2(principal),
			Interest:         round2(interest),
			Insurance:        round2(monthlyInsurance),
			RemainingBalance: round2(math.Max(0, remaining)),
		})
	}

	totalLoanCost := totalInterest + totalInsurance
	bankFees := input.BankFees
	guaranteeFees := input.GuaranteeFees
	brokerFees := input.BrokerFees
	renovationCost := input.RenovationCost
	totalProjectCost := input.PropertyPrice + notaryFees + agencyFees + bankFees + guaranteeFees + brokerFees + renovationCost + totalInterest + totalInsurance

	// Income comparison
	incomeMonthly := input.NetIncome1 + input.NetIncome2
	incomeTotal25y := incomeMonthly * 12 * 25
	var effortRate, projectIncomeRatio float64
	if incomeMonthly > 0 {
		effortRate = (monthlyPayment + monthlyInsurance) / incomeMonthly * 100
	}
	if incomeTotal25y > 0 {
		projectIncomeRatio = totalProjectCost / incomeTotal25y * 100
	}

	// Maximum borrowing capacity (HCSF 35% rule)
	// Max monthly payment (including insurance) = 35% of net income
	var maxMonthlyPayment, maxLoanAmount float64
	if incomeMonthly > 0 {
		maxMonthlyPayment = incomeMonthly * 0.35
		// Calculate max loan amount from max monthly payment
		// M + Insurance = maxMonthlyPayment
		// M = C × t / (1 - (1+t)^(-n))
		// Insurance = InsuranceRate / 100 / 12 × C
		// C × [t / (1 - (1+t)^(-n)) + InsuranceRate/100/12] = maxMonthlyPayment
		insuranceMonthlyRate := input.InsuranceRate / 100 / 12
		if avgMonthlyRate == 0 {
			// Without interest: M = C/n, so C = M × n
			// C/n + insuranceRate × C = maxMonthlyPayment
			// C × (1/n + insuranceRate) = maxMonthlyPayment
			divisor := 1/float64(n) + insuranceMonthlyRate
			if divisor > 0 {
				maxLoanAmount = maxMonthlyPayment / divisor
			}
		} else {
			paymentFactor := avgMonthlyRate / (1 - math.Pow(1+avgMonthlyRate, float64(-n)))
			divisor := paymentFactor + insuranceMonthlyRate
			if divisor > 0 {
				maxLoanAmount = maxMonthlyPayment / divisor
			}
		}
	}

	// Resale profitability projection (scénarios de valorisation annuelle)
	resaleRates := input.ResaleRates
	if len(resaleRates) == 0 {
		resaleRates = []float64{-0.01, 0, 0.01} // Valeurs par défaut
	}
	durationYears := n / 12
	downPayment := input.PropertyPrice - input.LoanAmount

	// Determine effective work lines: use WorkLines if provided, otherwise convert legacy data
	effectiveWorkLines := input.WorkLines
	if len(effectiveWorkLines) == 0 && renovationCost > 0 {
		// Legacy mode: convert to work lines using the custom rate
		renovationValueRate := input.RenovationValueRate
		if renovationValueRate == 0 {
			renovationValueRate = 70 // 70% par défaut si non renseigné
		}
		// Create a custom "legacy" work line that matches the old behavior
		effectiveWorkLines = []model.WorkLine{
			{
				CategoryID: "legacy",
				Label:      "Travaux (mode simple)",
				Amount:     renovationCost * renovationValueRate / 100, // Pre-apply the rate as amount
			},
		}
	}

	// Calculate initial work value (year 0) for baseValue at year 0
	initialWorkValue := CalculateTotalWorkValueAtYear(effectiveWorkLines, 0)

	resaleData := make([]model.ResaleProjection, 0, durationYears)
	saleCashData := make([]model.SaleCashProjection, 0, durationYears)

	// Tracking pour les points d'inflexion de rentabilité
	prevScenarios := make([]float64, len(resaleRates))
	resaleInflectionFound := make([]bool, len(resaleRates))
	resaleInflectionYears := make([]float64, len(resaleRates))
	for i := range resaleInflectionYears {
		resaleInflectionYears[i] = -1 // -1 = jamais rentable
	}

	// Variables pour le détail des coûts irrécupérables (fin de prêt)
	var finalCumulInterest, finalCumulInsurance, finalCumulCondoFees, finalCumulPropertyTax, finalCumulMaintenance float64

	// Frais d'entretien annuels (% de la valeur du bien)
	annualMaintenanceCost := input.PropertyPrice * input.MaintenanceRate / 100

	for year := 1; year <= durationYears; year++ {
		monthIndex := year*12 - 1 // index in amortization (0-based)
		capitalRemaining := amortization[monthIndex].RemainingBalance

		// Cumul des intérêts et assurance sur year*12 mois (coûts irrécupérables)
		var cumulInterest, cumulInsurance float64
		for m := 0; m < year*12; m++ {
			cumulInterest += amortization[m].Interest
			cumulInsurance += amortization[m].Insurance
		}

		// Coûts récurrents cumulés
		cumulCondoFees := input.CondoFees * float64(year*12)
		cumulPropertyTax := input.PropertyTax * float64(year)
		cumulMaintenance := annualMaintenanceCost * float64(year)

		// Coûts irrécupérables = frais fixes + intérêts + assurance + charges + taxe foncière + entretien
		// Le capital remboursé n'est PAS un coût irrécupérable (c'est une épargne forcée)
		irrecoverableCosts := notaryFees + agencyFees + bankFees + brokerFees + cumulInterest + cumulInsurance + cumulCondoFees + cumulPropertyTax + cumulMaintenance

		scenarios := make([]float64, len(resaleRates))
		// Calculate work value at this year (evolves over time based on category rates)
		workValueThisYear := CalculateTotalWorkValueAtYear(effectiveWorkLines, year)
		for i, rate := range resaleRates {
			// Les travaux valorisent le bien selon leur catégorie et évoluent dans le temps
			// La valeur du bien + travaux évolue avec le marché immobilier
			baseValue := input.PropertyPrice + workValueThisYear
			baseValueYear0 := input.PropertyPrice + initialWorkValue
			propertyValue := baseValue * math.Pow(1+rate, float64(year))
			// Plus-value nette = Appréciation du bien - Coûts irrécupérables
			appreciation := propertyValue - baseValueYear0
			netGain := appreciation - irrecoverableCosts
			scenarios[i] = round2(netGain)

			// Détection du point d'inflexion (rentabilité devient positive)
			if !resaleInflectionFound[i] && year > 1 {
				if prevScenarios[i] < 0 && scenarios[i] >= 0 {
					// Interpolation linéaire pour trouver l'année exacte
					if scenarios[i]-prevScenarios[i] != 0 {
						ratio := -prevScenarios[i] / (scenarios[i] - prevScenarios[i])
						resaleInflectionYears[i] = float64(year-1) + ratio
					} else {
						resaleInflectionYears[i] = float64(year)
					}
					resaleInflectionFound[i] = true
				}
			}
			prevScenarios[i] = scenarios[i]
		}

		resaleData = append(resaleData, model.ResaleProjection{
			Year:      year,
			Scenarios: scenarios,
		})

		// Sale Cash projection: cash récupéré à la revente
		// Coûts irrécupérables = frais fixes + intérêts cumulés + assurance cumulée + taxe foncière + charges copro + entretien
		// Note: cumulInterest, cumulInsurance, cumulCondoFees, cumulPropertyTax, cumulMaintenance déjà calculés ci-dessus
		irrecoverableCost := notaryFees + agencyFees + bankFees + brokerFees + cumulInterest + cumulInsurance + cumulCondoFees + cumulPropertyTax + cumulMaintenance

		grossCash := make([]float64, len(resaleRates))
		netCash := make([]float64, len(resaleRates))
		for i, rate := range resaleRates {
			// Les travaux valorisent le bien selon leur catégorie et évoluent dans le temps
			baseValue := input.PropertyPrice + workValueThisYear
			propertyValue := baseValue * math.Pow(1+rate, float64(year))
			// Frais de vente à la revente (agence, diagnostics, etc.)
			sellCosts := propertyValue * (input.ResaleSellCosts / 100)
			// Cash brut = Valeur bien - Frais de vente - Capital restant dû
			grossCash[i] = round2(propertyValue - sellCosts - capitalRemaining)
			// Cash net = Cash brut - Coûts irrécupérables
			netCash[i] = round2(grossCash[i] - irrecoverableCost)
		}

		saleCashData = append(saleCashData, model.SaleCashProjection{
			Year:              year,
			GrossCash:         grossCash,
			IrrecoverableCost: round2(irrecoverableCost),
			NetCash:           netCash,
		})

		// Sauvegarder les valeurs finales pour le détail des coûts irrécupérables
		if year == durationYears {
			finalCumulInterest = cumulInterest
			finalCumulInsurance = cumulInsurance
			finalCumulCondoFees = cumulCondoFees
			finalCumulPropertyTax = cumulPropertyTax
			finalCumulMaintenance = cumulMaintenance
		}
	}

	// Calcul étendu des points d'inflexion au-delà de la durée du prêt (jusqu'à 50 ans)
	// Après la fin du prêt : plus de mensualités, mais les charges continuent
	// Pré-calcul des intérêts/assurance totaux à la fin du prêt
	var totalInterestAtEnd, totalInsuranceAtEnd float64
	for m := 0; m < n; m++ {
		totalInterestAtEnd += amortization[m].Interest
		totalInsuranceAtEnd += amortization[m].Insurance
	}

	for i, rate := range resaleRates {
		if !resaleInflectionFound[i] {
			// Continuer le calcul au-delà de la durée du prêt
			for year := durationYears + 1; year <= 50; year++ {
				// Après le prêt : capital restant = 0, plus de mensualités
				cumulCondoFees := input.CondoFees * float64(year*12)
				cumulPropertyTax := input.PropertyTax * float64(year)
				cumulMaintenance := annualMaintenanceCost * float64(year)
				irrecoverableCosts := notaryFees + agencyFees + bankFees + brokerFees + totalInterestAtEnd + totalInsuranceAtEnd + cumulCondoFees + cumulPropertyTax + cumulMaintenance

				// Calculate work value at this year
				workValueExtended := CalculateTotalWorkValueAtYear(effectiveWorkLines, year)
				baseValue := input.PropertyPrice + workValueExtended
				propertyValue := baseValue * math.Pow(1+rate, float64(year))
				appreciation := propertyValue - (input.PropertyPrice + initialWorkValue)
				netGain := appreciation - irrecoverableCosts

				if prevScenarios[i] < 0 && netGain >= 0 {
					// Interpolation linéaire
					if netGain-prevScenarios[i] != 0 {
						ratio := -prevScenarios[i] / (netGain - prevScenarios[i])
						resaleInflectionYears[i] = float64(year-1) + ratio
					} else {
						resaleInflectionYears[i] = float64(year)
					}
					resaleInflectionFound[i] = true
					break
				}
				prevScenarios[i] = netGain
			}
		}
	}

	// Détail des coûts irrécupérables
	irrecoverableBreakdown := model.IrrecoverableDetail{
		NotaryFees:       round2(notaryFees),
		AgencyFees:       round2(agencyFees),
		BankFees:         round2(bankFees),
		BrokerFees:       round2(brokerFees),
		TotalInterest:    round2(finalCumulInterest),
		TotalInsurance:   round2(finalCumulInsurance),
		TotalCondoFees:   round2(finalCumulCondoFees),
		TotalPropertyTax: round2(finalCumulPropertyTax),
		TotalMaintenance: round2(finalCumulMaintenance),
		Total:            round2(notaryFees + agencyFees + bankFees + brokerFees + finalCumulInterest + finalCumulInsurance + finalCumulCondoFees + finalCumulPropertyTax + finalCumulMaintenance),
	}

	// Rent vs Buy comparison
	// Rent vs Buy comparison - Patrimoine net (plus haut = mieux)
	// Intègre les deux biais critiques:
	// 1. Biais de l'apport: l'apport placé génère des intérêts composés
	// 2. Biais du cash-flow: si mensualité > loyer, le locataire épargne la différence
	var rentVsBuyData []model.RentVsBuyYear
	if input.MonthlyRent > 0 {
		rentIncRate := input.RentIncreaseRate / 100
		savingsRate := input.SavingsRate / 100
		inflationRate := input.InflationRate / 100
		monthlyReturnRate := math.Pow(1+savingsRate, 1.0/12) - 1 // Taux mensuel équivalent
		rentVsBuyData = make([]model.RentVsBuyYear, 0, durationYears)

		// Coût mensuel total de l'acheteur (mensualité + assurance + taxe foncière + charges)
		buyerMonthlyCost := monthlyPayment + monthlyInsurance + (input.PropertyTax / 12) + input.CondoFees

		var cumulRent float64
		var cashFlowSavings float64 // Épargne cumulée de la différence mensuelle

		// Tracking pour le point d'inflexion
		prevRentWealth := make([]float64, len(resaleRates))
		prevBuyWealth := make([]float64, len(resaleRates))
		inflectionFound := make([]bool, len(resaleRates))
		inflectionYears := make([]float64, len(resaleRates))
		for i := range inflectionYears {
			inflectionYears[i] = -1 // -1 = jamais atteint
		}

		for year := 1; year <= durationYears; year++ {
			// Calcul mois par mois pour cette année
			for m := (year - 1) * 12; m < year*12; m++ {
				yearOfMonth := m / 12 // 0-indexed year
				rent := input.MonthlyRent * math.Pow(1+rentIncRate, float64(yearOfMonth))
				cumulRent += rent

				// Différence mensuelle: si le locataire paie moins, il peut épargner
				monthlySaving := buyerMonthlyCost - rent
				if monthlySaving > 0 {
					// L'épargne existante génère des intérêts ce mois
					cashFlowSavings *= (1 + monthlyReturnRate)
					// Puis on ajoute l'épargne du mois
					cashFlowSavings += monthlySaving
				} else {
					// Même si le loyer est plus cher, l'épargne existante génère des intérêts
					cashFlowSavings *= (1 + monthlyReturnRate)
				}
			}

			// Valeur de l'apport placé avec intérêts composés
			investmentValue := downPayment * math.Pow(1+savingsRate, float64(year))

			// Patrimoine locataire = apport placé + épargne cash-flow - loyers cumulés
			// Note: loyers déjà déduits via le cash-flow, donc on simplifie:
			// rentWealth = investmentValue + cashFlowSavings - cumulRent
			// Mais cashFlowSavings = cumul(buyerCost - rent) placé
			// Donc rentWealth = apport placé + (buyerCost - rent) placé = total épargné
			rentWealth := investmentValue + cashFlowSavings - cumulRent

			monthIndex := year*12 - 1
			capitalRemaining := amortization[monthIndex].RemainingBalance

			// Coûts irrécupérables pour l'acheteur
			var cumulInterest, cumulInsurance float64
			for m := 0; m < year*12; m++ {
				cumulInterest += amortization[m].Interest
				cumulInsurance += amortization[m].Insurance
			}
			cumulPropertyTax := input.PropertyTax * float64(year)
			cumulCondoFees := input.CondoFees * float64(year*12)
			cumulMaintenanceRvB := annualMaintenanceCost * float64(year)
			irrecoverableCosts := notaryFees + agencyFees + bankFees + brokerFees + cumulInterest + cumulInsurance + cumulPropertyTax + cumulCondoFees + cumulMaintenanceRvB

			buyWealth := make([]float64, len(resaleRates))
			// Calculate work value for rent vs buy comparison
			workValueRentVsBuy := CalculateTotalWorkValueAtYear(effectiveWorkLines, year)
			for i, rate := range resaleRates {
				// Les travaux valorisent le bien selon leur catégorie et évoluent dans le temps
				baseValue := input.PropertyPrice + workValueRentVsBuy
				propertyValue := baseValue * math.Pow(1+rate, float64(year))
				// Équité = valeur bien - capital restant dû
				equity := propertyValue - capitalRemaining
				// Patrimoine acheteur = équité - coûts irrécupérables
				buyWealth[i] = round2(equity - irrecoverableCosts)

				// Détection du point d'inflexion (achat devient > location)
				if !inflectionFound[i] && year > 1 {
					// Cherche l'intersection par interpolation linéaire
					if prevBuyWealth[i] <= prevRentWealth[i] && buyWealth[i] > rentWealth {
						// Interpolation: à quel moment exact les courbes se croisent?
						// prevYear + (crossover ratio)
						deltaBuy := buyWealth[i] - prevBuyWealth[i]
						deltaRent := rentWealth - prevRentWealth[i]
						gapPrev := prevRentWealth[i] - prevBuyWealth[i]
						if deltaBuy-deltaRent != 0 {
							ratio := gapPrev / (deltaBuy - deltaRent)
							inflectionYears[i] = float64(year-1) + ratio
						} else {
							inflectionYears[i] = float64(year)
						}
						inflectionFound[i] = true
					}
				}
				prevBuyWealth[i] = buyWealth[i]
			}
			for i := range prevRentWealth {
				prevRentWealth[i] = rentWealth
			}

			// Calcul de la mensualité en euros constants (pouvoir d'achat année 0)
			// Formule: RealPayment(year) = NominalPayment / (1 + inflationRate)^year
			realBuyerPayment := buyerMonthlyCost / math.Pow(1+inflationRate, float64(year))

			// Loyer nominal pour cette année
			rentThisYear := input.MonthlyRent * math.Pow(1+rentIncRate, float64(year-1))

			rentVsBuyData = append(rentVsBuyData, model.RentVsBuyYear{
				Year:             year,
				CumulRent:        round2(cumulRent),
				InvestmentValue:  round2(investmentValue),
				CashFlowSavings:  round2(cashFlowSavings),
				RentWealth:       round2(rentWealth),
				BuyWealth:        buyWealth,
				InflectionYears:  inflectionYears,
				NominalBuyerCost: round2(buyerMonthlyCost),
				RealBuyerPayment: round2(realBuyerPayment),
				Rent:             round2(rentThisYear),
			})
		}
	}

	// Property sale calculation (vente résidence principale)
	var propertySale model.PropertySale
	if input.CurrentSalePrice > 0 {
		propertySale.SalePrice = input.CurrentSalePrice
		propertySale.LoanBalance = input.CurrentLoanBalance
		propertySale.Penalty = input.EarlyRepaymentPenalty
		propertySale.LoanLines = input.CurrentLoanLines
		propertySale.NetProceeds = math.Max(0, input.CurrentSalePrice-input.CurrentLoanBalance-input.EarlyRepaymentPenalty)

		// Nouvelle logique équitable :
		// 1. Chacun récupère d'abord son apport initial
		// 2. Le bénéfice (ou la perte) est partagé selon le % convenu

		// Calcul des mois écoulés depuis le début du prêt pour E2
		now := time.Now()
		currentYear := now.Year()
		currentMonth := int(now.Month())
		loanStartYear := input.CurrentLoanStartYear
		loanStartMonth := input.CurrentLoanStartMonth
		if loanStartYear == 0 {
			loanStartYear = currentYear
			loanStartMonth = currentMonth
		}
		monthsElapsed := (currentYear-loanStartYear)*12 + (currentMonth - loanStartMonth)
		if monthsElapsed < 0 {
			monthsElapsed = 0
		}

		apportE1 := input.CurrentDownPayment1
		apportE2 := calculateE2AccumulatedContribution(input.VirtualContribution2, input.VirtualMonthlyPayment2, input.VirtualPaymentTiers2, monthsElapsed)
		totalApports := apportE1 + apportE2

		// Calcul du bénéfice (ou perte) = Produit net - Total des apports
		profit := propertySale.NetProceeds - totalApports

		// Part du bénéfice pour E2 selon le pourcentage convenu
		var profitShareE2 float64
		if profit > 0 {
			// Bénéfice : E2 reçoit son % du profit
			profitShareE2 = profit * input.VirtualProfitShare2 / 100
		} else {
			// Perte : E2 absorbe son % de la perte (réduction de son apport récupéré)
			profitShareE2 = profit * input.VirtualProfitShare2 / 100
		}

		// Calcul des montants finaux
		// E1 récupère : son apport + (profit - part E2)
		// E2 récupère : son apport + sa part du profit
		proceeds1 := apportE1 + (profit - profitShareE2)
		proceeds2 := apportE2 + profitShareE2

		// S'assurer qu'on ne dépasse pas le produit net disponible
		// et qu'on ne va pas en négatif
		if proceeds1 < 0 {
			proceeds2 += proceeds1 // Transférer le déficit
			proceeds1 = 0
		}
		if proceeds2 < 0 {
			proceeds1 += proceeds2 // Transférer le déficit
			proceeds2 = 0
		}

		propertySale.Proceeds1 = round2(proceeds1)
		propertySale.Proceeds2 = round2(proceeds2)

		// Stocker le détail des contributions pour l'affichage
		apportE2Monthly := calculateE2AccumulatedContribution(0, input.VirtualMonthlyPayment2, input.VirtualPaymentTiers2, monthsElapsed)
		apportE2Initial := input.VirtualContribution2

		// Calculer les versements de prêt E1 (mensualités cumulées) par ligne
		var apportE1Loans float64
		loanSchedule := CalculateCurrentLoanSchedule(input.CurrentLoanLines)

		// Structure pour accumuler le détail par ligne de prêt
		type loanAccum struct {
			total     float64
			principal float64
			interest  float64
			insurance float64
		}
		loanPaymentsByLine := make(map[string]*loanAccum)
		for _, line := range input.CurrentLoanLines {
			if line.Label != "" {
				loanPaymentsByLine[line.Label] = &loanAccum{}
			}
		}

		// Accumuler les versements mois par mois
		for m := 1; m <= monthsElapsed && m <= len(loanSchedule); m++ {
			apportE1Loans += loanSchedule[m-1].TotalAmount
			// Détail par ligne
			for _, payment := range loanSchedule[m-1].Payments {
				if payment.Label != "" {
					if accum, ok := loanPaymentsByLine[payment.Label]; ok {
						accum.total += payment.Total
						accum.principal += payment.Principal
						accum.interest += payment.Interest
						accum.insurance += payment.Insurance
					}
				}
			}
		}

		// Construire le slice de détail dans l'ordre des lignes d'entrée
		var apportE1LoansDetail []model.LoanPaymentDetail
		for _, line := range input.CurrentLoanLines {
			if line.Label != "" {
				if accum, ok := loanPaymentsByLine[line.Label]; ok && accum.total > 0 {
					apportE1LoansDetail = append(apportE1LoansDetail, model.LoanPaymentDetail{
						Label:     line.Label,
						Amount:    round2(accum.total),
						Principal: round2(accum.principal),
						Interest:  round2(accum.interest),
						Insurance: round2(accum.insurance),
					})
				}
			}
		}

		// E1 total versements = apport initial + versements prêt - remboursements E2
		apportE1TotalVersements := input.CurrentDownPayment1 + apportE1Loans - apportE2Monthly

		// Calculer les pourcentages de contribution (basés sur les apports effectifs, pas les versements)
		// Apport E1 = down payment (les versements de prêt ne sont pas des "apports" au sens de l'investissement)
		// Apport E2 = contribution initiale + mensualités cumulées
		var pctE1, pctE2 float64
		if totalApports > 0 {
			pctE1 = apportE1 / totalApports * 100
			pctE2 = apportE2 / totalApports * 100
		}

		propertySale.ApportE1Initial = round2(input.CurrentDownPayment1)
		propertySale.ApportE1Loans = round2(apportE1Loans)
		propertySale.ApportE1LoansDetail = apportE1LoansDetail
		propertySale.ApportE1Total = round2(apportE1TotalVersements)
		propertySale.ApportE2Initial = round2(apportE2Initial)
		propertySale.ApportE2Monthly = round2(apportE2Monthly)
		propertySale.ApportE2Total = round2(apportE2)
		propertySale.TotalApports = round2(totalApports)
		propertySale.ContributionPctE1 = round2(pctE1)
		propertySale.ContributionPctE2 = round2(pctE2)
		propertySale.MonthsElapsed = monthsElapsed
		propertySale.Profit = round2(profit)
		propertySale.ProfitShareE2 = round2(profitShareE2)
	}

	// Calculate current property projection if applicable
	var currentPropertyProjection []model.CurrentPropertyMonthProjection
	var currentLoanSchedule []model.MonthlySchedule
	var currentBorrowerPayments []model.CurrentBorrowerPayment
	if input.CurrentSalePrice > 0 && len(input.CurrentLoanLines) > 0 {
		currentPropertyProjection = calculateCurrentPropertyProjection(input, resaleRates)
		currentLoanSchedule = CalculateCurrentLoanSchedule(input.CurrentLoanLines)

		// Calculate cumulative payments per borrower for current property
		// Only if E2 has contributions (virtual contribution or monthly payments)
		if input.VirtualContribution2 > 0 || input.VirtualMonthlyPayment2 > 0 || len(input.VirtualPaymentTiers2) > 0 {
			// Determine max months: max of loan schedule and E2 payment tiers
			maxMonths := len(currentLoanSchedule)
			for _, tier := range input.VirtualPaymentTiers2 {
				if tier.EndMonth > maxMonths {
					maxMonths = tier.EndMonth
				}
			}

			currentBorrowerPayments = make([]model.CurrentBorrowerPayment, maxMonths)
			// E1 starts with initial down payment, then adds monthly loan payments
			// BUT we subtract E2's monthly reimbursements (E2 pays E1 to cover part of the loan)
			cumulLoanPayments := 0.0
			cumulE2MonthlyReimbursements := 0.0
			for month := 1; month <= maxMonths; month++ {
				// Add E1 loan payment if within loan schedule
				if month <= len(currentLoanSchedule) {
					cumulLoanPayments += currentLoanSchedule[month-1].TotalAmount
				}
				// E2's monthly reimbursements to E1 (excluding initial contribution)
				cumulE2MonthlyReimbursements = calculateE2AccumulatedContribution(0, input.VirtualMonthlyPayment2, input.VirtualPaymentTiers2, month)
				// E1 NET = down payment + loan payments - what E2 reimburses monthly
				cumulPaymentE1 := input.CurrentDownPayment1 + cumulLoanPayments - cumulE2MonthlyReimbursements
				// E2 NET = initial contribution + monthly reimbursements
				cumulPaymentE2 := input.VirtualContribution2 + cumulE2MonthlyReimbursements
				currentBorrowerPayments[month-1] = model.CurrentBorrowerPayment{
					Month:         month,
					CumulPayment1: round2(cumulPaymentE1),
					CumulPayment2: round2(cumulPaymentE2),
				}
			}
		}
	}

	// Ownership shares (quotes-parts) calculation
	var ownership model.OwnershipShare
	if input.DownPayment1 > 0 || input.DownPayment2 > 0 || propertySale.NetProceeds > 0 {
		monthlyTotal := monthlyPayment + monthlyInsurance
		if input.PaymentSplitMode == "equal" {
			// 50/50 split
			ownership.LoanShare1 = capital / 2
			ownership.LoanShare2 = capital / 2
			ownership.MonthlyPayment1 = monthlyTotal / 2
			ownership.MonthlyPayment2 = monthlyTotal / 2
		} else {
			// Prorata des revenus (default)
			totalIncome := input.NetIncome1 + input.NetIncome2
			if totalIncome > 0 {
				ratio1 := input.NetIncome1 / totalIncome
				ownership.LoanShare1 = capital * ratio1
				ownership.LoanShare2 = capital * (1 - ratio1)
				ownership.MonthlyPayment1 = monthlyTotal * ratio1
				ownership.MonthlyPayment2 = monthlyTotal * (1 - ratio1)
			} else {
				// No income info: 50/50 fallback
				ownership.LoanShare1 = capital / 2
				ownership.LoanShare2 = capital / 2
				ownership.MonthlyPayment1 = monthlyTotal / 2
				ownership.MonthlyPayment2 = monthlyTotal / 2
			}
		}
		// Include sale proceeds in contributions
		ownership.SaleProceeds1 = propertySale.Proceeds1
		ownership.SaleProceeds2 = propertySale.Proceeds2
		ownership.Contribution1 = input.DownPayment1 + propertySale.Proceeds1 + ownership.LoanShare1
		ownership.Contribution2 = input.DownPayment2 + propertySale.Proceeds2 + ownership.LoanShare2
		totalContribution := ownership.Contribution1 + ownership.Contribution2
		if totalContribution > 0 {
			ownership.QuotePart1 = round2(ownership.Contribution1 / totalContribution * 100)
			ownership.QuotePart2 = round2(ownership.Contribution2 / totalContribution * 100)
		}
		ownership.LoanShare1 = round2(ownership.LoanShare1)
		ownership.LoanShare2 = round2(ownership.LoanShare2)
		ownership.Contribution1 = round2(ownership.Contribution1)
		ownership.Contribution2 = round2(ownership.Contribution2)
		ownership.MonthlyPayment1 = round2(ownership.MonthlyPayment1)
		ownership.MonthlyPayment2 = round2(ownership.MonthlyPayment2)
	}

	// Calculate aid eligibility (PTZ, PAL, BRS)
	aidEligibility := CalculateAidEligibility(input)

	// Loyer équivalent = coûts irrécupérables récurrents / durée
	// On inclut les coûts récurrents (intérêts, assurance, taxe foncière, charges, entretien)
	// mais PAS les frais ponctuels (notaire, agence, dossier) car ils ne sont pas mensuels
	equivalentRent := (totalInterest + totalInsurance +
		input.PropertyTax*float64(durationYears) +
		input.CondoFees*float64(n) +
		annualMaintenanceCost*float64(durationYears)) / float64(n)

	// TRI (Taux de Rendement Interne) calculation per year and scenario
	// Allows comparing the real estate investment with financial placements
	irrData := make([]model.IRRProjection, 0, durationYears)
	irrByScenarioFinal := make([]float64, len(resaleRates))

	// Initial investment (year 0 outflow)
	// Includes: down payment + notary fees + agency fees + bank fees + guarantee fees + renovation cost
	initialInvestment := downPayment + notaryFees + agencyFees + bankFees + guaranteeFees + brokerFees + renovationCost

	// Annual cost during loan period (same every year)
	// Includes: loan payments + insurance + property tax + condo fees + maintenance
	annualLoanPayment := (monthlyPayment + monthlyInsurance) * 12
	annualPropertyTax := input.PropertyTax
	annualCondoFees := input.CondoFees * 12
	// Note: annualMaintenanceCost already computed above

	// Rent increase rate for rent savings calculation (same as rent vs buy comparison)
	rentIncRate := input.RentIncreaseRate / 100

	for year := 1; year <= durationYears; year++ {
		irrs := make([]float64, len(resaleRates))
		for i := range resaleRates {
			// Build cash flow array
			cashFlows := make([]float64, year+1)
			cashFlows[0] = -initialInvestment

			// Annual costs for years 1 to year-1, offset by rent savings (with inflation)
			for t := 1; t < year; t++ {
				rentSavings := input.MonthlyRent * 12 * math.Pow(1+rentIncRate, float64(t-1))
				cashFlows[t] = -(annualLoanPayment + annualPropertyTax + annualCondoFees + annualMaintenanceCost) + rentSavings
			}

			// Final year: annual cost + rent savings + sale proceeds (gross cash from saleCashData)
			saleProceeds := saleCashData[year-1].GrossCash[i]
			rentSavingsFinal := input.MonthlyRent * 12 * math.Pow(1+rentIncRate, float64(year-1))
			cashFlows[year] = -(annualLoanPayment + annualPropertyTax + annualCondoFees + annualMaintenanceCost) + rentSavingsFinal + saleProceeds

			irr := calculateIRR(cashFlows)
			if math.IsNaN(irr) {
				irrs[i] = -999 // Indicator for non-convergence
			} else {
				irrs[i] = round2(irr * 100) // Convert to %
			}
		}
		irrData = append(irrData, model.IRRProjection{Year: year, IRR: irrs})

		if year == durationYears {
			irrByScenarioFinal = irrs
		}
	}

	// Calculate tier-based monthly schedule if multiple loan lines
	var monthlySchedule []model.MonthlySchedule
	if len(input.NewLoanLines) > 0 {
		monthlySchedule = calculateTierBasedSchedule(input.NewLoanLines, input.DurationMonths)
	}

	// Calculate energy comparison
	energyComparisonData := CalculateEnergyComparison(input, durationYears)

	// Calculate bridge loan if enabled
	var bridgeLoan model.BridgeLoanResult
	if input.BridgeLoanEnabled && input.CurrentSalePrice > 0 {
		bridgeLoan = CalculateBridgeLoan(
			input.CurrentSalePrice,
			input.BridgeLoanQuotity,
			input.BridgeLoanRate,
			input.BridgeLoanDuration,
			input.BridgeLoanInsurance,
			input.BridgeLoanFranchise,
			input.CurrentLoanBalance,
		)
		// Bridge loan cost adds to total project cost
		totalProjectCost += bridgeLoan.TotalCost
		// Bridge loan monthly payment impacts effort rate during relay period
		if incomeMonthly > 0 && bridgeLoan.MonthlyPayment > 0 {
			effortRate = (monthlyPayment + monthlyInsurance + bridgeLoan.MonthlyPayment) / incomeMonthly * 100
		}
	}

	return model.CreditResult{
		MonthlyPayment:   round2(monthlyPayment),
		MonthlyInsurance: round2(monthlyInsurance),
		MonthlyTotal:     round2(monthlyPayment + monthlyInsurance),
		TotalInterest:    round2(totalInterest),
		TotalInsurance:   round2(totalInsurance),
		TotalLoanCost:    round2(totalLoanCost),
		NotaryFees:       round2(notaryFees),
		AgencyFees:       round2(agencyFees),
		BankFees:         round2(bankFees),
		GuaranteeFees:    round2(guaranteeFees),
		BrokerFees:       round2(brokerFees),
		RenovationCost:   round2(renovationCost),
		TotalProjectCost: round2(totalProjectCost),
		IncomeMonthly:      round2(incomeMonthly),
		IncomeTotal25y:     round2(incomeTotal25y),
		EffortRate:         round2(effortRate),
		ProjectIncomeRatio: round2(projectIncomeRatio),
		MaxMonthlyPayment:  round2(maxMonthlyPayment),
		MaxLoanAmount:      round2(maxLoanAmount),
		Amortization:      amortization,
		ResaleData:             resaleData,
		ResaleRates:            resaleRates,
		ResaleInflectionYears:  resaleInflectionYears,
		IrrecoverableBreakdown: irrecoverableBreakdown,
		RentVsBuyData:          rentVsBuyData,
		SaleCashData:     saleCashData,
		IRRData:          irrData,
		IRRByScenario:    irrByScenarioFinal,
		Ownership:                  ownership,
		PropertySale:               propertySale,
		CurrentPropertyProjection:  currentPropertyProjection,
		AidEligibility:             aidEligibility,
		LoanLineResults:         loanLineResults,
		EquivalentRent:          round2(equivalentRent),
		MonthlySchedule:         monthlySchedule,
		CurrentLoanSchedule:      currentLoanSchedule,
		CurrentBorrowerPayments:  currentBorrowerPayments,
		EnergyComparisonData:     energyComparisonData,
		BridgeLoan:               bridgeLoan,
	}
}

// CalculateBridgeLoan computes the results for a bridge loan (prêt relais).
// Franchise partielle: monthly interest payments, capital repaid in fine at sale.
// Franchise totale: no monthly payments, interest capitalized, everything repaid at sale.
func CalculateBridgeLoan(salePrice, quotity, rate float64, durationMonths int, insuranceRate float64, franchise string, loanBalance float64) model.BridgeLoanResult {
	amount := round2(salePrice * quotity / 100)
	if amount <= 0 || durationMonths <= 0 {
		return model.BridgeLoanResult{}
	}

	monthlyRate := rate / 100 / 12
	monthlyInsuranceRate := insuranceRate / 100 / 12

	var monthlyPayment, totalInterest, totalInsurance, capitalizedAmount float64

	if franchise == "totale" {
		// Franchise totale: no monthly payment, interest capitalized
		monthlyPayment = 0
		capitalizedAmount = amount * math.Pow(1+monthlyRate, float64(durationMonths))
		totalInterest = capitalizedAmount - amount
		// Insurance still accrues monthly on original amount
		totalInsurance = monthlyInsuranceRate * amount * float64(durationMonths)
		// Capitalized amount includes insurance
		capitalizedAmount += totalInsurance
	} else {
		// Franchise partielle: monthly interest + insurance, capital repaid in fine
		monthlyInterest := amount * monthlyRate
		monthlyIns := amount * monthlyInsuranceRate
		monthlyPayment = round2(monthlyInterest + monthlyIns)
		totalInterest = monthlyInterest * float64(durationMonths)
		totalInsurance = monthlyIns * float64(durationMonths)
		capitalizedAmount = amount // Only capital to repay at sale
	}

	totalCost := totalInterest + totalInsurance

	netAmount := amount - loanBalance
	if netAmount < 0 {
		netAmount = 0
	}

	return model.BridgeLoanResult{
		Enabled:           true,
		Amount:            amount,
		NetAmount:         round2(netAmount),
		Rate:              rate,
		Duration:          durationMonths,
		Franchise:         franchise,
		MonthlyPayment:    round2(monthlyPayment),
		TotalInterest:     round2(totalInterest),
		TotalInsurance:    round2(totalInsurance),
		TotalCost:         round2(totalCost),
		CapitalizedAmount: round2(capitalizedAmount),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// npv calculates the Net Present Value at a given rate
func npv(cashFlows []float64, rate float64) float64 {
	result := 0.0
	for t, cf := range cashFlows {
		result += cf / math.Pow(1+rate, float64(t))
	}
	return result
}

// calculateIRR computes the Internal Rate of Return for a series of cash flows.
// cashFlows[0] is the initial investment (negative), cashFlows[1..n] are annual flows.
// Returns IRR as a decimal (0.05 = 5%). Returns NaN if no convergence.
func calculateIRR(cashFlows []float64) float64 {
	const maxIterations = 100
	const tolerance = 1e-7

	// Try Newton-Raphson with multiple initial guesses
	initialGuesses := []float64{0.1, 0.0, -0.1, 0.2, -0.2, 0.05, -0.05}

	for _, guess := range initialGuesses {
		r := guess
		converged := true

		for i := 0; i < maxIterations; i++ {
			npvVal := 0.0
			dnpv := 0.0
			for t, cf := range cashFlows {
				discount := math.Pow(1+r, float64(t))
				if discount == 0 {
					converged = false
					break
				}
				npvVal += cf / discount
				if t > 0 {
					dnpv -= float64(t) * cf / (discount * (1 + r))
				}
			}

			if !converged {
				break
			}

			if math.Abs(npvVal) < tolerance {
				return r
			}

			if math.Abs(dnpv) < 1e-10 {
				converged = false
				break
			}

			newR := r - npvVal/dnpv

			// Bounds check - IRR typically between -50% and 100%
			if newR < -0.5 || newR > 1.0 {
				converged = false
				break
			}

			r = newR
		}

		if converged {
			return r
		}
	}

	// Fallback to bisection method
	low := -0.5
	high := 1.0

	// Find bounds where NPV changes sign
	npvLow := npv(cashFlows, low)
	npvHigh := npv(cashFlows, high)

	// If both have same sign, try to find better bounds
	if npvLow*npvHigh > 0 {
		// Expand search range
		for _, testRate := range []float64{-0.9, -0.3, 0.5, 2.0} {
			testNPV := npv(cashFlows, testRate)
			if testNPV*npvLow < 0 {
				high = testRate
				npvHigh = testNPV
				break
			}
			if testNPV*npvHigh < 0 {
				low = testRate
				npvLow = testNPV
				break
			}
		}
	}

	// If still same sign, no IRR exists in reasonable range
	if npvLow*npvHigh > 0 {
		return math.NaN()
	}

	// Bisection
	for i := 0; i < maxIterations; i++ {
		mid := (low + high) / 2
		npvMid := npv(cashFlows, mid)

		if math.Abs(npvMid) < tolerance {
			return mid
		}

		if npvMid*npvLow < 0 {
			high = mid
			npvHigh = npvMid
		} else {
			low = mid
			npvLow = npvMid
		}
	}

	return (low + high) / 2 // Return best approximation
}

// ComputeLoanRemainingBalance computes the remaining balance and monthly payment for a LoanLine
// at the given date (year, month). Returns remainingBalance, amortizedCapital, monthlyPayment, monthlyInsurance.
func ComputeLoanRemainingBalance(line model.LoanLine, atYear, atMonth int) (remaining, amortized, monthly, insurance float64) {
	if line.OriginalAmount <= 0 {
		return 0, 0, 0, 0
	}

	// Si l'utilisateur a saisi un CRD (Balance), l'utiliser pour remaining/amortized
	// mais calculer la mensualité à partir du montant initial (elle ne change pas)
	userBalance := line.Balance

	// Derive total duration from tiers if DurationYears is not set
	totalDurationMonths := line.DurationYears*12 + line.DeferralMonths
	if line.DurationYears <= 0 && len(line.Tiers) > 0 {
		// Infer duration from last tier's EndMonth + deferral
		lastEnd := 0
		for _, t := range line.Tiers {
			if t.EndMonth > lastEnd {
				lastEnd = t.EndMonth
			}
		}
		totalDurationMonths = lastEnd + line.DeferralMonths
	} else if line.DurationYears <= 0 {
		return 0, 0, 0, 0
	}
	monthlyRate := line.Rate / 100 / 12

	// Deferral rate
	deferralRate := line.DeferralRate
	if deferralRate == 0 {
		deferralRate = line.Rate
	}
	monthlyDeferralRate := deferralRate / 100 / 12

	// Compute months elapsed since loan start
	if line.StartYear == 0 {
		// No start date: can't compute elapsed months, return full balance
		return line.OriginalAmount, 0, 0, 0
	}
	startMonths := line.StartYear*12 + line.StartMonth
	currentMonths := atYear*12 + atMonth
	monthsElapsed := currentMonths - startMonths
	if monthsElapsed < 0 {
		monthsElapsed = 0
	}
	if monthsElapsed > totalDurationMonths {
		monthsElapsed = totalDurationMonths
	}

	balance := line.OriginalAmount

	// If tiers are defined, use them; otherwise compute constant payment
	if len(line.Tiers) > 0 {
		for m := 1; m <= monthsElapsed; m++ {
			if balance <= 0 {
				break
			}
			if m <= line.DeferralMonths {
				// Deferral: interest only, no principal
				continue
			}
			interest := balance * monthlyRate
			amortMonth := m - line.DeferralMonths
			tierPayment := getPaymentForMonth(line.Tiers, amortMonth)
			principal := tierPayment - interest
			if principal < 0 {
				principal = 0
			}
			if principal > balance {
				principal = balance
			}
			balance -= principal
		}
		// Monthly payment: use current tier
		if monthsElapsed <= line.DeferralMonths {
			monthly = balance * monthlyDeferralRate
		} else {
			amortMonth := monthsElapsed - line.DeferralMonths + 1
			monthly = getPaymentForMonth(line.Tiers, amortMonth)
		}
	} else {
		// Constant payment amortization
		amortDuration := totalDurationMonths - line.DeferralMonths
		if amortDuration <= 0 {
			return line.OriginalAmount, 0, 0, 0
		}
		if monthlyRate == 0 {
			monthly = line.OriginalAmount / float64(amortDuration)
		} else {
			monthly = line.OriginalAmount * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-amortDuration)))
		}

		for m := 1; m <= monthsElapsed; m++ {
			if balance <= 0 {
				break
			}
			if m <= line.DeferralMonths {
				continue
			}
			interest := balance * monthlyRate
			principal := monthly - interest
			if principal > balance {
				principal = balance
			}
			balance -= principal
		}
	}

	if balance < 0.01 {
		balance = 0
	}

	// Inclure l'assurance dans la mensualité pour cohérence avec le graphique
	var monthlyInsurance float64
	if line.InsuranceMonthly > 0 {
		monthlyInsurance = line.InsuranceMonthly
	} else {
		monthlyInsurance = line.InsuranceRate / 100 / 12 * line.OriginalAmount
	}
	monthly += monthlyInsurance

	// Si l'utilisateur a saisi un CRD, utiliser cette valeur pour remaining/amortized
	// mais garder la mensualité calculée (elle est basée sur le montant initial)
	if userBalance > 0 {
		remaining = round2(userBalance)
		amortized = round2(line.OriginalAmount - userBalance)
	} else {
		remaining = round2(balance)
		amortized = round2(line.OriginalAmount - balance)
	}
	monthly = round2(monthly)
	insurance = round2(monthlyInsurance)
	return
}

// ComputeMonthlyPaymentAt returns the total monthly payment (principal + interest + insurance)
// for a LoanLine at a given absolute month (1-based from loan start).
// It handles deferral periods, tiers, and constant-payment amortization.
func ComputeMonthlyPaymentAt(line model.LoanLine, absoluteMonth int) float64 {
	if line.OriginalAmount <= 0 || absoluteMonth <= 0 {
		return 0
	}

	totalDurationMonths := line.DurationYears*12 + line.DeferralMonths
	if line.DurationYears <= 0 && len(line.Tiers) > 0 {
		lastEnd := 0
		for _, t := range line.Tiers {
			if t.EndMonth > lastEnd {
				lastEnd = t.EndMonth
			}
		}
		totalDurationMonths = lastEnd + line.DeferralMonths
	}
	if totalDurationMonths <= 0 {
		return 0
	}
	if absoluteMonth > totalDurationMonths {
		return 0
	}

	monthlyRate := line.Rate / 100 / 12
	deferralRate := line.DeferralRate
	if deferralRate == 0 {
		deferralRate = line.Rate
	}
	monthlyDeferralRate := deferralRate / 100 / 12

	// Insurance
	var monthlyInsurance float64
	if line.InsuranceMonthly > 0 {
		monthlyInsurance = line.InsuranceMonthly
	} else {
		monthlyInsurance = line.InsuranceRate / 100 / 12 * line.OriginalAmount
	}

	// During deferral: interest only + insurance
	if absoluteMonth <= line.DeferralMonths {
		return round2(line.OriginalAmount*monthlyDeferralRate + monthlyInsurance)
	}

	// Post-deferral: need to simulate balance up to this month
	amortMonth := absoluteMonth - line.DeferralMonths
	balance := line.OriginalAmount

	if len(line.Tiers) > 0 {
		// Tier-based: simulate to find balance, then return tier payment + insurance
		for m := 1; m < amortMonth; m++ {
			interest := balance * monthlyRate
			tierPayment := getPaymentForMonth(line.Tiers, m)
			principal := tierPayment - interest
			if principal < 0 {
				principal = 0
			}
			if principal > balance {
				principal = balance
			}
			balance -= principal
			if balance < 0.01 {
				balance = 0
				break
			}
		}
		if balance <= 0 {
			return round2(monthlyInsurance)
		}
		tierPayment := getPaymentForMonth(line.Tiers, amortMonth)
		return round2(tierPayment + monthlyInsurance)
	}

	// Constant payment amortization
	amortDuration := totalDurationMonths - line.DeferralMonths
	if amortDuration <= 0 {
		return round2(monthlyInsurance)
	}
	var monthlyPayment float64
	if monthlyRate == 0 {
		monthlyPayment = line.OriginalAmount / float64(amortDuration)
	} else {
		monthlyPayment = line.OriginalAmount * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-amortDuration)))
	}
	return round2(monthlyPayment + monthlyInsurance)
}

// PTZ income ceilings by zone and household size (2024 barème)
// Index: household size - 1 (0 = 1 person, 4 = 5+ persons)
var ptzIncomeCeilings = map[string][]float64{
	"A":    {49000, 73500, 88200, 102900, 117600},
	"Abis": {49000, 73500, 88200, 102900, 117600},
	"B1":   {34500, 51750, 62100, 72450, 82800},
	"B2":   {31500, 47250, 56700, 66150, 75600},
	"C":    {28500, 42750, 51300, 59850, 68400},
}

// PTZ maximum amount percentages by zone (% of property price, new housing)
var ptzMaxPercentage = map[string]float64{
	"A":    0.50,
	"Abis": 0.50,
	"B1":   0.40,
	"B2":   0.20,
	"C":    0.20,
}

// PAL (Prêt Action Logement) income ceilings by zone and household size (2024)
var palIncomeCeilings = map[string][]float64{
	"A":    {39363, 58831, 77174, 92159, 107049},
	"Abis": {43475, 64958, 85175, 101693, 118146},
	"B1":   {39363, 58831, 77174, 92159, 107049},
	"B2":   {32814, 49042, 64312, 76799, 89208},
	"C":    {32814, 49042, 64312, 76799, 89208},
}

// calculateCurrentPropertyProjection computes the E1/E2 split projection for the current property over time.
func calculateCurrentPropertyProjection(input model.CreditInput, rates []float64) []model.CurrentPropertyMonthProjection {
	// Date actuelle pour calculer les mois écoulés
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// 1. Calculer la mensualité et durée restante pour chaque ligne de prêt
	type loanAmort struct {
		balance         float64
		monthlyPayment  float64
		monthlyRate     float64
		remainingMonths int
	}
	loans := make([]loanAmort, len(input.CurrentLoanLines))

	maxMonths := 0
	totalBalance := 0.0
	for i, line := range input.CurrentLoanLines {
		if line.Balance <= 0 {
			continue
		}
		monthlyRate := line.Rate / 100 / 12

		// Calculer les mois restants à partir de la date de début et durée
		remainingMonths := 20 * 12 // Défaut 20 ans
		if line.DurationYears > 0 {
			totalDuration := line.DurationYears * 12
			if line.StartYear > 0 {
				// Mois écoulés depuis le début du prêt
				startMonths := line.StartYear*12 + line.StartMonth
				currentMonths := currentYear*12 + currentMonth
				elapsedMonths := currentMonths - startMonths
				if elapsedMonths < 0 {
					elapsedMonths = 0
				}
				remainingMonths = totalDuration - elapsedMonths
				if remainingMonths < 0 {
					remainingMonths = 0
				}
			} else {
				// Pas de date de début, utiliser la durée totale
				remainingMonths = totalDuration
			}
		}

		if remainingMonths > maxMonths {
			maxMonths = remainingMonths
		}

		var monthlyPayment float64
		if monthlyRate == 0 || remainingMonths == 0 {
			if remainingMonths > 0 {
				monthlyPayment = line.Balance / float64(remainingMonths)
			}
		} else {
			monthlyPayment = line.Balance * monthlyRate / (1 - math.Pow(1+monthlyRate, -float64(remainingMonths)))
		}

		loans[i] = loanAmort{
			balance:         line.Balance,
			monthlyPayment:  monthlyPayment,
			monthlyRate:     monthlyRate,
			remainingMonths: remainingMonths,
		}
		totalBalance += line.Balance
	}

	// Calculer le nombre de mois à projeter (durée max restante, plafonnée à 25 ans)
	if maxMonths > 25*12 {
		maxMonths = 25 * 12
	}
	if maxMonths == 0 {
		return nil // Tous les prêts sont terminés
	}

	// 3. IRA total
	totalIRA := input.EarlyRepaymentPenalty

	// 4. Projeter mois par mois
	projections := make([]model.CurrentPropertyMonthProjection, 0, maxMonths)

	// Garder le solde courant pour chaque prêt (calcul incrémental)
	currentBalances := make([]float64, len(loans))
	for i := range loans {
		currentBalances[i] = loans[i].balance
	}

	for month := 1; month <= maxMonths; month++ {
		// Calculer le CRD après ce mois (incrémental)
		monthBalance := 0.0
		for i := range loans {
			if currentBalances[i] <= 0 {
				continue
			}
			interest := currentBalances[i] * loans[i].monthlyRate
			principal := loans[i].monthlyPayment - interest
			currentBalances[i] -= principal
			if currentBalances[i] < 0 {
				currentBalances[i] = 0
			}
			monthBalance += currentBalances[i]
		}

		// Valeur du bien et parts par scénario
		propertyValues := make([]float64, len(rates))
		proceeds1 := make([]float64, len(rates))
		proceeds2 := make([]float64, len(rates))

		for i, rate := range rates {
			// Valorisation du bien avec taux annuel converti en mois
			propertyValues[i] = input.CurrentSalePrice * math.Pow(1+rate, float64(month)/12.0)

			// Calcul du partage E1/E2
			netProceeds := propertyValues[i] - monthBalance - totalIRA
			if netProceeds < 0 {
				netProceeds = 0
			}

			// Contribution totale E2 = apport initial + mensualités cumulées (avec paliers si définis)
			// E2's monthly payments reimburse E1, so we need to calculate E1's NET contribution
			e2MonthlyReimbursements := calculateE2AccumulatedContribution(0, input.VirtualMonthlyPayment2, input.VirtualPaymentTiers2, month)
			principalRepaid := totalBalance - monthBalance
			// E1 NET = down payment + principal repaid - what E2 reimbursed monthly
			apportE1 := input.CurrentDownPayment1 + principalRepaid - e2MonthlyReimbursements
			// E2 = initial contribution + monthly reimbursements
			apportE2 := input.VirtualContribution2 + e2MonthlyReimbursements
			totalApports := apportE1 + apportE2

			// E2's share is proportional to their actual contribution
			var shareE2 float64
			if totalApports > 0 {
				shareE2 = apportE2 / totalApports
			}

			p1 := netProceeds * (1 - shareE2)
			p2 := netProceeds * shareE2

			// Éviter les valeurs négatives
			if p1 < 0 {
				p2 += p1
				p1 = 0
			}
			if p2 < 0 {
				p1 += p2
				p2 = 0
			}

			proceeds1[i] = round2(p1)
			proceeds2[i] = round2(p2)
		}

		projections = append(projections, model.CurrentPropertyMonthProjection{
			Month:         month,
			PropertyValue: propertyValues,
			LoanBalance:   round2(monthBalance),
			Proceeds1:     proceeds1,
			Proceeds2:     proceeds2,
		})

		// Arrêter si le prêt est remboursé
		if monthBalance <= 0 {
			break
		}
	}

	return projections
}

// CalculateCurrentLoanSchedule computes the month-by-month payment schedule for existing loans.
// Uses manual tiers defined by the user for each loan line, with deferral support.
func CalculateCurrentLoanSchedule(lines []model.LoanLine) []model.MonthlySchedule {
	if len(lines) == 0 {
		return nil
	}

	// currentLoanState tracks display info for a single existing loan
	type currentLoanState struct {
		label            string
		totalMonths      int
		deferralMonths   int
		originalAmount   float64
		balance          float64
		monthlyRate      float64 // Taux mensuel normal
		deferralRate     float64 // Taux mensuel pour intérêts intercalaires
		monthlyInsurance float64 // Assurance mensuelle
		tiers            []model.PaymentTier
		constantPayment  float64 // Fallback quand pas de paliers
	}

	// Create a state for EVERY line to keep alignment with input array
	states := make([]*currentLoanState, len(lines))
	var maxTotalMonths int

	for i, line := range lines {
		// Determine total months from durationYears + deferralMonths or max tier endMonth
		totalMonths := 0
		if line.DurationYears > 0 {
			totalMonths = line.DurationYears*12 + line.DeferralMonths
		}

		// Also check max endMonth from tiers
		// Tiers represent post-deferral amortization months, so add deferralMonths
		for _, tier := range line.Tiers {
			tierTotal := tier.EndMonth + line.DeferralMonths
			if tierTotal > totalMonths {
				totalMonths = tierTotal
			}
		}

		// Default if nothing set
		if totalMonths == 0 {
			totalMonths = 20 * 12
		}

		if totalMonths > maxTotalMonths {
			maxTotalMonths = totalMonths
		}

		// Taux d'intérêts intercalaires
		deferralRate := line.DeferralRate
		if deferralRate == 0 {
			deferralRate = line.Rate
		}

		// Assurance mensuelle : utiliser InsuranceMonthly si défini, sinon calculer à partir du taux
		var monthlyInsurance float64
		if line.InsuranceMonthly > 0 {
			monthlyInsurance = line.InsuranceMonthly
		} else {
			monthlyInsurance = line.InsuranceRate / 100 / 12 * line.OriginalAmount
		}

		monthlyRate := line.Rate / 100 / 12
		var constantPayment float64
		amortDuration := totalMonths - line.DeferralMonths
		if amortDuration > 0 {
			if monthlyRate == 0 {
				constantPayment = line.OriginalAmount / float64(amortDuration)
			} else {
				constantPayment = line.OriginalAmount * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-amortDuration)))
			}
		}

		states[i] = &currentLoanState{
			label:            line.Label,
			totalMonths:      totalMonths,
			deferralMonths:   line.DeferralMonths,
			originalAmount:   line.OriginalAmount,
			balance:          line.OriginalAmount,
			monthlyRate:      monthlyRate,
			deferralRate:     deferralRate / 100 / 12,
			monthlyInsurance: monthlyInsurance,
			tiers:            line.Tiers,
			constantPayment:  constantPayment,
		}
	}

	if maxTotalMonths == 0 {
		return nil
	}

	schedule := make([]model.MonthlySchedule, 0, maxTotalMonths)

	for month := 1; month <= maxTotalMonths; month++ {
		payments := make([]model.LoanMonthPayment, len(states))
		var monthTotal float64

		for i, s := range states {
			payments[i] = model.LoanMonthPayment{Label: s.label}

			if month > s.totalMonths || s.balance <= 0 {
				continue
			}

			var tierPayment, interest, principal, insurance float64

			// Pendant le différé : intérêts intercalaires seulement
			if month <= s.deferralMonths {
				interest = s.balance * s.deferralRate
				tierPayment = interest
				principal = 0
				insurance = s.monthlyInsurance
			} else {
				// Get payment from tier - offset by deferral months
				amortizationMonth := month - s.deferralMonths
				if len(s.tiers) > 0 {
					tierPayment = getPaymentForMonth(s.tiers, amortizationMonth)
				} else {
					tierPayment = s.constantPayment
				}

				// Calculer intérêts et capital
				interest = s.balance * s.monthlyRate
				insurance = s.monthlyInsurance

				// Le paiement tier inclut capital + intérêts (pas l'assurance)
				// Donc: principal = tierPayment - interest
				principal = tierPayment - interest
				if principal < 0 {
					principal = 0
				}
				if principal > s.balance {
					principal = s.balance
				}

				// Mettre à jour le solde
				s.balance -= principal
				if s.balance < 0.01 {
					s.balance = 0
				}
			}

			total := tierPayment + insurance

			payments[i] = model.LoanMonthPayment{
				Label:     s.label,
				Principal: round2(principal),
				Interest:  round2(interest),
				Insurance: round2(insurance),
				Total:     round2(total),
			}
			monthTotal += total
		}

		schedule = append(schedule, model.MonthlySchedule{
			Month:       month,
			Payments:    payments,
			TotalAmount: round2(monthTotal),
		})
	}

	return schedule
}

// CalculateAidEligibility computes eligibility for PTZ, PAL, and BRS.
func CalculateAidEligibility(input model.CreditInput) model.AidEligibility {
	result := model.AidEligibility{}

	// Sum RFRs from both borrowers
	rfrYear1Total := input.RFRYear1_1 + input.RFRYear1_2
	rfrYear2Total := input.RFRYear2_1 + input.RFRYear2_2

	// Determine reference RFR (use max of N-1 and N-2 totals)
	referenceRFR := math.Max(rfrYear1Total, rfrYear2Total)
	result.PTZReferenceRFR = referenceRFR

	// Household size index (capped at 5+ persons)
	householdIndex := input.HouseholdSize - 1
	if householdIndex < 0 {
		householdIndex = 0
	}
	if householdIndex > 4 {
		householdIndex = 4
	}

	// Get zone-specific ceilings
	zone := input.PropertyZone
	if zone == "" {
		zone = "B1" // Default zone
	}

	// PTZ eligibility
	ptzCeilings, ptzOK := ptzIncomeCeilings[zone]
	if ptzOK && householdIndex < len(ptzCeilings) {
		result.PTZIncomeCeiling = ptzCeilings[householdIndex]
		result.PTZEligible = referenceRFR > 0 && referenceRFR <= result.PTZIncomeCeiling

		if result.PTZEligible {
			// Calculate max PTZ amount
			maxPercentage := ptzMaxPercentage[zone]
			// PTZ is capped at a percentage of the property price (including renovation)
			baseAmount := input.PropertyPrice + input.RenovationCost
			result.PTZMaxAmount = round2(baseAmount * maxPercentage)

			// Apply PTZ ceiling limits (varies by zone and household size)
			// Simplified: use a general cap of 150,000€ for zones A/Abis/B1
			var ptzCap float64
			switch zone {
			case "A", "Abis":
				ptzCap = 150000
			case "B1":
				ptzCap = 135000
			case "B2":
				ptzCap = 110000
			case "C":
				ptzCap = 100000
			}
			if result.PTZMaxAmount > ptzCap {
				result.PTZMaxAmount = ptzCap
			}
		}
	}

	// PAL eligibility
	palCeilings, palOK := palIncomeCeilings[zone]
	if palOK && householdIndex < len(palCeilings) {
		palCeiling := palCeilings[householdIndex]
		result.PALEligible = referenceRFR > 0 && referenceRFR <= palCeiling

		if result.PALEligible {
			// PAL max amount is 40,000€ (2024)
			result.PALMaxAmount = 40000
		}
	}

	// BRS eligibility (uses same income ceilings as PAL/social housing)
	// BRS is typically available in tense zones (A, Abis, B1)
	if zone == "A" || zone == "Abis" || zone == "B1" {
		if palOK && householdIndex < len(palCeilings) {
			brsCeiling := palCeilings[householdIndex]
			result.BRSEligible = referenceRFR > 0 && referenceRFR <= brsCeiling
		}
	}

	return result
}

// estimateKWh estimates annual kWh consumption from cost when kWh is not provided.
// Uses default French energy prices: gas ~0.08 €/kWh, electricity ~0.2267 €/kWh.
// If property 1 has both cost and kWh data for that energy type, uses that ratio instead.
func estimateKWh(kWh, cost, refKWh, refCost, defaultPrice float64) float64 {
	if kWh > 0 {
		return kWh
	}
	if cost <= 0 {
		return 0
	}
	// Use property 1's €/kWh if available
	if refKWh > 0 && refCost > 0 {
		return round2(cost * refKWh / refCost)
	}
	return round2(cost / defaultPrice)
}

// CalculateEnergyComparison computes the cumulative energy costs for up to three properties over time.
// It takes into account annual price increases for both gas and electricity.
// Costs are expected as annual values (€/year).
func CalculateEnergyComparison(input model.CreditInput, durationYears int) []model.EnergyComparisonYear {
	// Skip if no energy data provided
	hasEnergyCosts := input.Energy1Gas != 0 || input.Energy1Electricity != 0 || input.Energy1Other != 0 ||
		input.Energy2Gas != 0 || input.Energy2Electricity != 0 || input.Energy2Other != 0 ||
		input.Energy3Gas != 0 || input.Energy3Electricity != 0 || input.Energy3Other != 0
	if !hasEnergyCosts {
		return nil
	}

	priceIncrease := input.EnergyPriceIncrease / 100

	// Default French energy prices (€/kWh)
	const defaultGasPrice = 0.08
	const defaultElecPrice = 0.2267

	// Estimate kWh from costs when not explicitly provided
	gasKWh1 := estimateKWh(input.Energy1GasKWh, input.Energy1Gas, 0, 0, defaultGasPrice)
	elecKWh1 := estimateKWh(input.Energy1ElectricityKWh, input.Energy1Electricity, 0, 0, defaultElecPrice)
	gasKWh2 := estimateKWh(input.Energy2GasKWh, input.Energy2Gas, input.Energy1GasKWh, input.Energy1Gas, defaultGasPrice)
	elecKWh2 := estimateKWh(input.Energy2ElectricityKWh, input.Energy2Electricity, input.Energy1ElectricityKWh, input.Energy1Electricity, defaultElecPrice)
	gasKWh3 := estimateKWh(input.Energy3GasKWh, input.Energy3Gas, input.Energy1GasKWh, input.Energy1Gas, defaultGasPrice)
	elecKWh3 := estimateKWh(input.Energy3ElectricityKWh, input.Energy3Electricity, input.Energy1ElectricityKWh, input.Energy1Electricity, defaultElecPrice)

	data := make([]model.EnergyComparisonYear, 0, durationYears)

	var cumulGas1, cumulElec1, cumulOther1 float64
	var cumulGas2, cumulElec2, cumulOther2 float64
	var cumulGas3, cumulElec3, cumulOther3 float64
	var cumulGasKWh1, cumulElecKWh1, cumulGasKWh2, cumulElecKWh2 float64
	var cumulGasKWh3, cumulElecKWh3 float64

	for year := 1; year <= durationYears; year++ {
		// Price multiplier for this year (increases each year)
		priceMultiplier := math.Pow(1+priceIncrease, float64(year-1))

		// Annual costs for this year with price increase applied
		annualGas1 := input.Energy1Gas * priceMultiplier
		annualElec1 := input.Energy1Electricity * priceMultiplier
		annualOther1 := input.Energy1Other * priceMultiplier
		annualGas2 := input.Energy2Gas * priceMultiplier
		annualElec2 := input.Energy2Electricity * priceMultiplier
		annualOther2 := input.Energy2Other * priceMultiplier
		annualGas3 := input.Energy3Gas * priceMultiplier
		annualElec3 := input.Energy3Electricity * priceMultiplier
		annualOther3 := input.Energy3Other * priceMultiplier

		// Add annual costs
		cumulGas1 += annualGas1
		cumulElec1 += annualElec1
		cumulOther1 += annualOther1
		cumulGas2 += annualGas2
		cumulElec2 += annualElec2
		cumulOther2 += annualOther2
		cumulGas3 += annualGas3
		cumulElec3 += annualElec3
		cumulOther3 += annualOther3

		// kWh tracking (no price increase, just consumption)
		cumulGasKWh1 += gasKWh1
		cumulElecKWh1 += elecKWh1
		cumulGasKWh2 += gasKWh2
		cumulElecKWh2 += elecKWh2
		cumulGasKWh3 += gasKWh3
		cumulElecKWh3 += elecKWh3

		cumulTotal1 := cumulGas1 + cumulElec1 + cumulOther1
		cumulTotal2 := cumulGas2 + cumulElec2 + cumulOther2
		cumulTotal3 := cumulGas3 + cumulElec3 + cumulOther3
		cumulSavings := cumulTotal1 - cumulTotal2 // Positive = property 2 is cheaper

		data = append(data, model.EnergyComparisonYear{
			Year:          year,
			CumulGas1:     round2(cumulGas1),
			CumulElec1:    round2(cumulElec1),
			CumulOther1:   round2(cumulOther1),
			CumulTotal1:   round2(cumulTotal1),
			CumulGas2:     round2(cumulGas2),
			CumulElec2:    round2(cumulElec2),
			CumulOther2:   round2(cumulOther2),
			CumulTotal2:   round2(cumulTotal2),
			CumulGas3:     round2(cumulGas3),
			CumulElec3:    round2(cumulElec3),
			CumulOther3:   round2(cumulOther3),
			CumulTotal3:   round2(cumulTotal3),
			CumulSavings:  round2(cumulSavings),
			CumulGasKWh1:  round2(cumulGasKWh1),
			CumulElecKWh1: round2(cumulElecKWh1),
			CumulGasKWh2:  round2(cumulGasKWh2),
			CumulElecKWh2: round2(cumulElecKWh2),
			CumulGasKWh3:  round2(cumulGasKWh3),
			CumulElecKWh3: round2(cumulElecKWh3),
		})
	}

	return data
}
