package model

// CreditInput holds the parameters for a mortgage simulation.
type CreditInput struct {
	PropertyPrice    float64 // Prix du bien
	LoanAmount       float64 // Montant emprunté
	InterestRate     float64 // Taux d'intérêt annuel (%)
	DurationMonths   int     // Durée en mois
	InsuranceRate    float64 // Taux assurance annuel (%)
	NotaryRate       float64 // Taux frais de notaire (%)
	AgencyRate       float64 // Taux frais d'agence (%)
	AgencyFixed      float64 // Frais d'agence montant fixe (alternatif)
	BankFees         float64 // Frais de dossier bancaire (€)
	StartYear        int     // Année de début
	StartMonth       int     // Mois de début (1-12)
	NetIncome1       float64 // Revenu mensuel net emprunteur 1
	NetIncome2       float64 // Revenu mensuel net emprunteur 2
	MonthlyRent      float64 // Loyer mensuel actuel
	RentIncreaseRate float64 // Revalorisation annuelle du loyer (%)
	SavingsRate      float64 // Taux de rendement épargne annuel (%) - coût d'opportunité
	PropertyTax      float64 // Taxe foncière annuelle (€)
	CondoFees            float64 // Charges de copropriété mensuelles (€)
	RenovationCost       float64 // Travaux immédiats (€)
	RenovationValueRate  float64 // Coefficient de valorisation des travaux (%) - ex: 70% = 1€ travaux = 0.70€ de valeur
}

// CreditResult holds the computed results of a mortgage simulation.
type CreditResult struct {
	MonthlyPayment   float64 // Mensualité hors assurance
	MonthlyInsurance float64 // Mensualité assurance
	MonthlyTotal     float64 // Mensualité totale
	TotalInterest    float64 // Coût total des intérêts
	TotalInsurance   float64 // Coût total assurance
	TotalLoanCost    float64 // Coût total du crédit
	NotaryFees       float64 // Frais de notaire
	AgencyFees       float64 // Frais d'agence
	BankFees         float64 // Frais de dossier
	RenovationCost   float64 // Travaux immédiats
	TotalProjectCost float64 // Coût total du projet
	IncomeMonthly    float64 // Revenus mensuels nets combinés
	IncomeTotal25y   float64 // Revenus totaux sur 25 ans
	EffortRate         float64 // Taux d'effort (mensualité / revenus %)
	ProjectIncomeRatio float64 // Coût projet / revenus 25 ans (%)
	MaxMonthlyPayment  float64 // Mensualité max selon règle HCSF 35%
	MaxLoanAmount      float64 // Capacité d'emprunt max selon règle HCSF 35%
	Amortization     []AmortizationRow
	ResaleData            []ResaleProjection   // Plus-value nette par année et scénario
	ResaleRates           []float64            // Taux de valorisation utilisés (ex: -0.01, 0, 0.01)
	ResaleInflectionYears []float64            // Point d'inflexion par scénario (année où achat devient rentable, -1 si jamais)
	IrrecoverableBreakdown IrrecoverableDetail // Détail des coûts irrécupérables
	RentVsBuyData    []RentVsBuyYear      // Données comparaison location vs achat
	SaleCashData     []SaleCashProjection // Cash récupéré à la revente par année
}

// ResaleProjection holds the net gain/loss for each valuation scenario at a given year.
type ResaleProjection struct {
	Year      int
	Scenarios []float64 // Plus-value nette pour chaque taux (même ordre que ResaleRates)
}

// RentVsBuyYear holds the comparison data for one year.
type RentVsBuyYear struct {
	Year                int
	CumulRent           float64   // Cumul des loyers
	InvestmentValue     float64   // Valeur de l'apport placé (avec intérêts composés)
	CashFlowSavings     float64   // Épargne cumulée de la différence mensuelle (mensualité - loyer)
	RentWealth          float64   // Patrimoine locataire = placement apport + épargne cash-flow - loyers cumulés
	BuyWealth           []float64 // Patrimoine acheteur par scénario = équité - coûts irrécupérables
	InflectionYears     []float64 // Point d'inflexion par scénario (année où achat > location, -1 si jamais)
}

// SaleCashProjection holds the cash recovered from selling at each year.
type SaleCashProjection struct {
	Year              int
	GrossCash         []float64 // Cash brut par scénario (valeur bien - capital restant)
	IrrecoverableCost float64   // Coûts irrécupérables cumulés
	NetCash           []float64 // Cash net par scénario (cash brut - coûts irrécupérables)
}

// IrrecoverableDetail holds the breakdown of irrecoverable costs.
type IrrecoverableDetail struct {
	NotaryFees   float64 // Frais de notaire
	AgencyFees   float64 // Frais d'agence
	BankFees     float64 // Frais de dossier
	TotalInterest float64 // Intérêts cumulés
	TotalInsurance float64 // Assurance cumulée
	TotalCondoFees float64 // Charges copro cumulées
	Total         float64 // Total des coûts irrécupérables
}

// AmortizationRow represents one month in the amortization schedule.
type AmortizationRow struct {
	Month            int
	Date             string  // Date formatée (ex: "Février 2026")
	Payment          float64 // Mensualité (capital + intérêts)
	Principal        float64 // Part capital
	Interest         float64 // Part intérêts
	Insurance        float64 // Assurance
	RemainingBalance float64 // Capital restant dû
}
