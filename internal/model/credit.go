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
	GuaranteeFees    float64 // Frais de garantie (hypothèque, caution, PPD) (€)
	StartYear        int     // Année de début
	StartMonth       int     // Mois de début (1-12)
	NetIncome1       float64 // Revenu mensuel net emprunteur 1
	NetIncome2       float64 // Revenu mensuel net emprunteur 2
	MonthlyRent      float64 // Loyer mensuel actuel
	RentIncreaseRate float64 // Revalorisation annuelle du loyer (%)
	SavingsRate      float64 // Taux de rendement épargne annuel (%) - coût d'opportunité
	InflationRate    float64 // Taux d'inflation annuel (%)
	PropertyTax      float64 // Taxe foncière annuelle (€)
	CondoFees            float64 // Charges de copropriété mensuelles (€)
	MaintenanceRate      float64 // Taux d'entretien annuel (% de la valeur du bien)
	RenovationCost       float64    // Travaux immédiats (€) - Mode simple (rétrocompatibilité)
	RenovationValueRate  float64    // Coefficient de valorisation des travaux (%) - Mode simple
	WorkLines            []WorkLine // Lignes de travaux détaillées avec évolution temporelle
	DownPayment1         float64    // Apport emprunteur 1 (€)
	DownPayment2     float64 // Apport emprunteur 2 (€)
	PaymentSplitMode string  // Mode de répartition des mensualités ("prorata" ou "equal")
	CurrentSalePrice      float64    // Prix de vente estimé du bien actuel (€)
	CurrentLoanBalance    float64    // Capital restant dû du prêt en cours (€) - total de toutes les lignes
	CurrentLoanLines      []LoanLine // Détail des lignes de prêt
	EarlyRepaymentPenalty float64    // Indemnités de remboursement anticipé - IRA (€) - total de toutes les lignes
	CurrentDownPayment1   float64    // Apport initial emprunteur 1 sur le bien actuel (€)
	SalePropertyShare1    float64 // Quote-part du bien actuel - Emprunteur 1 (%)
	VirtualContribution2   float64 // Contribution non-officielle de l'emprunteur 2 au bien actuel (€)
	VirtualProfitShare2    float64 // Part du bénéfice pour l'emprunteur 2 (%)
	VirtualMonthlyPayment2 float64 // Participation mensuelle virtuelle E2 (€)
	RFRYear2_1            float64 // RFR N-2 Emprunteur 1 (€)
	RFRYear1_1            float64 // RFR N-1 Emprunteur 1 (€)
	RFRYear2_2            float64 // RFR N-2 Emprunteur 2 (€)
	RFRYear1_2            float64 // RFR N-1 Emprunteur 2 (€)
	HouseholdSize         int            // Nombre de personnes dans le foyer fiscal
	PropertyZone          string         // Zone géographique (A, Abis, B1, B2, C)
	NewLoanLines          []NewLoanLine  // Lignes de crédit du nouveau prêt (PTZ, PAL, etc.)
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
	GuaranteeFees    float64 // Frais de garantie
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
	IRRData          []IRRProjection      // TRI par année et par scénario
	IRRByScenario    []float64            // TRI annuel final par scénario (%, même ordre que ResaleRates)
	Ownership        OwnershipShare       // Quotes-parts de propriété entre co-acheteurs
	PropertySale                PropertySale                    // Résultat de la vente du bien actuel
	CurrentPropertyProjection   []CurrentPropertyYearProjection // Projection bien actuel (E1/E2)
	AidEligibility              AidEligibility                  // Éligibilité aux aides (PTZ, PAL, BRS)
	LoanLineResults   []NewLoanLineResult   // Détail des résultats par ligne de crédit
	EquivalentRent    float64               // Loyer équivalent mensuel (coûts irrécupérables récurrents / mois)
}

// AidEligibility holds the eligibility results for housing assistance programs.
type AidEligibility struct {
	PTZEligible      bool    // Éligible au PTZ
	PTZMaxAmount     float64 // Montant max PTZ (€)
	PTZIncomeCeiling float64 // Plafond de ressources applicable (€)
	PTZReferenceRFR  float64 // RFR de référence utilisé (€)
	PALEligible      bool    // Éligible au Prêt Action Logement
	PALMaxAmount     float64 // Montant max PAL (€)
	BRSEligible      bool    // Éligible au BRS
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
	NominalBuyerCost    float64   // Coût mensuel nominal acheteur (mensualité + charges)
	RealBuyerPayment    float64   // Mensualité en euros constants (pouvoir d'achat année 0)
	Rent                float64   // Loyer nominal pour cette année
}

// SaleCashProjection holds the cash recovered from selling at each year.
type SaleCashProjection struct {
	Year              int
	GrossCash         []float64 // Cash brut par scénario (valeur bien - capital restant)
	IrrecoverableCost float64   // Coûts irrécupérables cumulés
	NetCash           []float64 // Cash net par scénario (cash brut - coûts irrécupérables)
}

// CurrentPropertyYearProjection holds one year's projection for the current property E1/E2 split.
type CurrentPropertyYearProjection struct {
	Year          int
	PropertyValue []float64 // Valeur du bien par scénario (-1%, 0%, +1%)
	LoanBalance   float64   // Capital restant dû total
	Proceeds1     []float64 // Part E1 par scénario
	Proceeds2     []float64 // Part E2 par scénario
}

// IRRProjection holds the IRR at each year for each scenario.
type IRRProjection struct {
	Year int
	IRR  []float64 // TRI pour chaque scénario (-1%, 0%, +1%)
}

// IrrecoverableDetail holds the breakdown of irrecoverable costs.
type IrrecoverableDetail struct {
	NotaryFees       float64 // Frais de notaire
	AgencyFees       float64 // Frais d'agence
	BankFees         float64 // Frais de dossier
	TotalInterest    float64 // Intérêts cumulés
	TotalInsurance   float64 // Assurance cumulée
	TotalCondoFees   float64 // Charges copro cumulées
	TotalPropertyTax float64 // Taxe foncière cumulée
	TotalMaintenance float64 // Frais d'entretien cumulés
	Total            float64 // Total des coûts irrécupérables
}

// PropertySale holds the result of selling the current property.
type PropertySale struct {
	SalePrice   float64    // Prix de vente
	LoanBalance float64    // Capital restant dû (total)
	Penalty     float64    // IRA (total)
	NetProceeds float64    // Produit net de vente
	Proceeds1   float64    // Part emprunteur 1
	Proceeds2   float64    // Part emprunteur 2
	LoanLines   []LoanLine // Détail des lignes de prêt
}

// OwnershipShare holds the ownership split between two co-buyers.
type OwnershipShare struct {
	MonthlyPayment1 float64 // Mensualité totale emprunteur 1
	MonthlyPayment2 float64 // Mensualité totale emprunteur 2
	LoanShare1      float64 // Part du prêt emprunteur 1
	LoanShare2      float64 // Part du prêt emprunteur 2
	SaleProceeds1   float64 // Produit vente emprunteur 1
	SaleProceeds2   float64 // Produit vente emprunteur 2
	Contribution1   float64 // Apport1 + SaleProceeds1 + LoanShare1
	Contribution2   float64 // Apport2 + SaleProceeds2 + LoanShare2
	QuotePart1      float64 // % propriété emprunteur 1
	QuotePart2      float64 // % propriété emprunteur 2
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

// LoanLine represents a single loan line for IRA calculation.
type LoanLine struct {
	Label         string  `json:"label"`         // Libellé du prêt (ex: "Prêt principal", "PTZ")
	Balance       float64 `json:"balance"`       // Capital restant dû
	Rate          float64 `json:"rate"`          // Taux d'intérêt (%)
	IRA           float64 `json:"ira"`           // IRA saisi par l'utilisateur
	StartYear     int     `json:"startYear"`     // Année de début du prêt
	StartMonth    int     `json:"startMonth"`    // Mois de début (1-12)
	DurationYears int     `json:"durationYears"` // Durée totale en années
}

// NewLoanLine represents a loan line for the new mortgage.
type NewLoanLine struct {
	Label         string  `json:"label"`         // Libellé du prêt (ex: "Prêt principal", "PTZ", "PAL")
	Amount        float64 `json:"amount"`        // Montant emprunté (€)
	Rate          float64 `json:"rate"`          // Taux d'intérêt annuel (%)
	DurationYears int     `json:"durationYears"` // Durée en années
	InsuranceRate float64 `json:"insuranceRate"` // Taux assurance annuel (%)
}

// NewLoanLineResult holds computed results for a single loan line.
type NewLoanLineResult struct {
	Label            string  // Libellé du prêt
	Amount           float64 // Montant emprunté
	Rate             float64 // Taux d'intérêt
	DurationYears    int     // Durée
	InsuranceRate    float64 // Taux assurance
	MonthlyPayment   float64 // Mensualité hors assurance
	MonthlyInsurance float64 // Mensualité assurance
	MonthlyTotal     float64 // Mensualité totale
	TotalInterest    float64 // Coût total des intérêts
	TotalInsurance   float64 // Coût total assurance
}

// WorkCategory represents a category of renovation work with its valuation parameters.
type WorkCategory struct {
	ID          string  `json:"id"`          // Identifiant unique (ex: "structure", "peinture")
	Label       string  `json:"label"`       // Libellé affiché (ex: "Gros œuvre / Structure")
	InitialRate float64 `json:"initialRate"` // Coefficient de valorisation immédiate (%)
	AnnualRate  float64 `json:"annualRate"`  // Taux de dépréciation (-) ou appréciation (+) par an (%)
}

// WorkLine represents a single line of renovation work.
type WorkLine struct {
	CategoryID string  `json:"categoryID"` // ID de la catégorie (ex: "structure")
	Label      string  `json:"label"`      // Libellé personnalisé (ex: "Refaire la toiture")
	Amount     float64 `json:"amount"`     // Montant des travaux (€)
}
