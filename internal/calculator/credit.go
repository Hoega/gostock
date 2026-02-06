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

// Calculate computes the full mortgage simulation from the given input.
func Calculate(input model.CreditInput) model.CreditResult {
	monthlyRate := input.InterestRate / 100 / 12
	n := input.DurationMonths
	capital := input.LoanAmount

	// Monthly payment (annuité constante): M = C × t / (1 - (1+t)^(-n))
	var monthlyPayment float64
	if monthlyRate == 0 {
		monthlyPayment = capital / float64(n)
	} else {
		monthlyPayment = capital * monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-n)))
	}

	// Insurance: annual rate on initial capital, divided by 12
	monthlyInsurance := input.InsuranceRate / 100 / 12 * capital

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
	remaining := capital
	var totalInterest float64
	amortization := make([]model.AmortizationRow, 0, n)

	for m := 1; m <= n; m++ {
		// Compute date for this row
		month := (startMonth-1+m-1)%12 + 1
		year := startYear + (startMonth-1+m-1)/12
		date := fmt.Sprintf("%s %d", frenchMonths[month-1], year)

		interest := remaining * monthlyRate
		principal := monthlyPayment - interest
		remaining -= principal

		// Avoid floating point noise on the last month
		if m == n {
			principal += remaining
			remaining = 0
		}

		totalInterest += interest

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

	totalInsurance := monthlyInsurance * float64(n)
	totalLoanCost := totalInterest + totalInsurance
	bankFees := input.BankFees
	renovationCost := input.RenovationCost
	totalProjectCost := input.PropertyPrice + notaryFees + agencyFees + bankFees + renovationCost + totalInterest + totalInsurance

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
		if monthlyRate == 0 {
			// Without interest: M = C/n, so C = M × n
			// C/n + insuranceRate × C = maxMonthlyPayment
			// C × (1/n + insuranceRate) = maxMonthlyPayment
			divisor := 1/float64(n) + insuranceMonthlyRate
			if divisor > 0 {
				maxLoanAmount = maxMonthlyPayment / divisor
			}
		} else {
			paymentFactor := monthlyRate / (1 - math.Pow(1+monthlyRate, float64(-n)))
			divisor := paymentFactor + insuranceMonthlyRate
			if divisor > 0 {
				maxLoanAmount = maxMonthlyPayment / divisor
			}
		}
	}

	// Resale profitability projection (scénarios de valorisation annuelle)
	resaleRates := []float64{-0.01, 0, 0.01}
	durationYears := n / 12
	downPayment := input.PropertyPrice - input.LoanAmount
	// Valeur ajoutée par les travaux (coefficient de valorisation, ex: 70% par défaut)
	renovationValueRate := input.RenovationValueRate / 100
	if renovationValueRate == 0 {
		renovationValueRate = 0.70 // 70% par défaut si non renseigné
	}
	renovationAddedValue := renovationCost * renovationValueRate
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
	var finalCumulInterest, finalCumulInsurance, finalCumulCondoFees float64

	for year := 1; year <= durationYears; year++ {
		monthIndex := year*12 - 1 // index in amortization (0-based)
		capitalRemaining := amortization[monthIndex].RemainingBalance

		// Cumul des mensualités sur year*12 mois
		var cumulPayments float64
		for m := 0; m < year*12; m++ {
			cumulPayments += amortization[m].Payment + amortization[m].Insurance
		}

		// Coûts récurrents cumulés (charges copro)
		cumulCondoFees := input.CondoFees * float64(year*12)

		totalSpent := downPayment + notaryFees + agencyFees + bankFees + renovationCost + cumulPayments + cumulCondoFees

		scenarios := make([]float64, len(resaleRates))
		for i, rate := range resaleRates {
			// Les travaux valorisent le bien selon le coefficient (ex: 70%)
			// 1€ de travaux ≠ 1€ de valeur (toiture vs cuisine)
			baseValue := input.PropertyPrice + renovationAddedValue
			propertyValue := baseValue * math.Pow(1+rate, float64(year))
			netGain := propertyValue - capitalRemaining - totalSpent
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
		// Coûts irrécupérables = frais fixes + intérêts cumulés + assurance cumulée + taxe foncière + charges copro
		var cumulInterest, cumulInsurance float64
		for m := 0; m < year*12; m++ {
			cumulInterest += amortization[m].Interest
			cumulInsurance += amortization[m].Insurance
		}
		irrecoverableCost := notaryFees + agencyFees + bankFees + cumulInterest + cumulInsurance + cumulCondoFees

		grossCash := make([]float64, len(resaleRates))
		netCash := make([]float64, len(resaleRates))
		for i, rate := range resaleRates {
			// Les travaux valorisent le bien selon le coefficient
			baseValue := input.PropertyPrice + renovationAddedValue
			propertyValue := baseValue * math.Pow(1+rate, float64(year))
			// Cash brut = Valeur bien - Capital restant dû
			grossCash[i] = round2(propertyValue - capitalRemaining)
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
		}
	}

	// Calcul étendu des points d'inflexion au-delà de la durée du prêt (jusqu'à 50 ans)
	// Après la fin du prêt : plus de mensualités, mais les charges continuent
	totalPaymentsAtEnd := float64(0)
	for m := 0; m < n; m++ {
		totalPaymentsAtEnd += amortization[m].Payment + amortization[m].Insurance
	}

	for i, rate := range resaleRates {
		if !resaleInflectionFound[i] {
			// Continuer le calcul au-delà de la durée du prêt
			for year := durationYears + 1; year <= 50; year++ {
				// Après le prêt : capital restant = 0, plus de mensualités
				cumulCondoFees := input.CondoFees * float64(year*12)
				totalSpent := downPayment + notaryFees + agencyFees + bankFees + renovationCost + totalPaymentsAtEnd + cumulCondoFees

				baseValue := input.PropertyPrice + renovationAddedValue
				propertyValue := baseValue * math.Pow(1+rate, float64(year))
				netGain := propertyValue - 0 - totalSpent // capitalRemaining = 0 après le prêt

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
		NotaryFees:     round2(notaryFees),
		AgencyFees:     round2(agencyFees),
		BankFees:       round2(bankFees),
		TotalInterest:  round2(finalCumulInterest),
		TotalInsurance: round2(finalCumulInsurance),
		TotalCondoFees: round2(finalCumulCondoFees),
		Total:          round2(notaryFees + agencyFees + bankFees + finalCumulInterest + finalCumulInsurance + finalCumulCondoFees),
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
			irrecoverableCosts := notaryFees + agencyFees + bankFees + cumulInterest + cumulInsurance + cumulPropertyTax + cumulCondoFees

			buyWealth := make([]float64, len(resaleRates))
			for i, rate := range resaleRates {
				// Les travaux valorisent le bien selon le coefficient
				baseValue := input.PropertyPrice + renovationAddedValue
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

			rentVsBuyData = append(rentVsBuyData, model.RentVsBuyYear{
				Year:            year,
				CumulRent:       round2(cumulRent),
				InvestmentValue: round2(investmentValue),
				CashFlowSavings: round2(cashFlowSavings),
				RentWealth:      round2(rentWealth),
				BuyWealth:       buyWealth,
				InflectionYears: inflectionYears,
			})
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
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
