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
	BrokerFees       float64 // Frais de courtage (€)
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
	CurrentLoanStartYear  int        // Année de début des prêts du bien actuel
	CurrentLoanStartMonth int        // Mois de début (1-12)
	EarlyRepaymentPenalty float64    // Indemnités de remboursement anticipé - IRA (€) - total de toutes les lignes
	CurrentDownPayment1   float64    // Apport initial emprunteur 1 sur le bien actuel (€)
	SalePropertyShare1    float64 // Quote-part du bien actuel - Emprunteur 1 (%)
	VirtualContribution2   float64 // Contribution non-officielle de l'emprunteur 2 au bien actuel (€)
	VirtualProfitShare2    float64       // Part du bénéfice pour l'emprunteur 2 (%)
	VirtualMonthlyPayment2 float64       // Participation mensuelle virtuelle E2 (€)
	VirtualPaymentTiers2   []PaymentTier // Paliers de participation E2
	RFRYear2_1            float64 // RFR N-2 Emprunteur 1 (€)
	RFRYear1_1            float64 // RFR N-1 Emprunteur 1 (€)
	RFRYear2_2            float64 // RFR N-2 Emprunteur 2 (€)
	RFRYear1_2            float64 // RFR N-1 Emprunteur 2 (€)
	HouseholdSize         int            // Nombre de personnes dans le foyer fiscal
	PropertyZone          string         // Zone géographique (A, Abis, B1, B2, C)
	NewLoanLines          []NewLoanLine  // Lignes de crédit du nouveau prêt (PTZ, PAL, etc.)
	// Energy comparison fields
	Energy1Gas            float64 // Gaz annuel bien 1 (€/an)
	Energy1Electricity    float64 // Électricité annuelle bien 1 (€/an)
	Energy1GasKWh         float64 // Consommation gaz bien 1 (kWh/an)
	Energy1ElectricityKWh float64 // Consommation électricité bien 1 (kWh/an)
	Energy1Other          float64 // Autres coûts annuels bien 1 (€/an) - ex: recharge VE
	Energy1OtherLabel     string  // Label autres coûts bien 1
	Energy1Label          string  // Label bien 1 (ex: "Bien actuel")
	Energy2Gas            float64 // Gaz annuel bien 2 (€/an)
	Energy2Electricity    float64 // Électricité annuelle bien 2 (€/an)
	Energy2GasKWh         float64 // Consommation gaz bien 2 (kWh/an)
	Energy2ElectricityKWh float64 // Consommation électricité bien 2 (kWh/an)
	Energy2Other          float64 // Autres coûts annuels bien 2 (€/an) - ex: recharge VE
	Energy2OtherLabel     string  // Label autres coûts bien 2
	Energy2Label          string  // Label bien 2 (ex: "Nouveau bien")
	Energy1Surface        float64 // Surface bien 1 (m²)
	Energy1DPE            float64 // DPE bien 1 (kWh/m²/an)
	Energy2Surface        float64 // Surface bien 2 (m²)
	Energy2DPE            float64 // DPE bien 2 (kWh/m²/an)
	Energy3Gas            float64 // Coût gaz annuel bien 3 (€/an)
	Energy3Electricity    float64 // Coût électricité annuel bien 3 (€/an)
	Energy3GasKWh         float64 // Consommation gaz bien 3 (kWh/an)
	Energy3ElectricityKWh float64 // Consommation électricité bien 3 (kWh/an)
	Energy3Other          float64 // Autres coûts énergétiques bien 3 (€/an)
	Energy3OtherLabel     string  // Label autres coûts bien 3
	Energy3Label          string  // Label bien 3
	Energy3Surface        float64 // Surface bien 3 (m²)
	Energy3DPE            float64 // DPE bien 3 (kWh/m²/an)
	EnergyPriceIncrease   float64 // Augmentation annuelle prix énergie (%)
	// Resale projection parameters
	ResaleRates     []float64 // Taux de revalorisation annuelle personnalisés (ex: [-0.02, 0, 0.03])
	ResaleSellCosts float64   // Frais de vente à la revente (%)
	// Prêt relais
	BridgeLoanEnabled   bool    // Activer le prêt relais
	BridgeLoanQuotity   float64 // Quotité bancaire (50-80%)
	BridgeLoanRate      float64 // Taux d'intérêt du prêt relais (%)
	BridgeLoanDuration  int     // Durée en mois (12-24)
	BridgeLoanInsurance float64 // Taux assurance (%)
	BridgeLoanFranchise string  // "partielle" (intérêts mensuels) ou "totale" (intérêts capitalisés)
	BridgeLoanSaleMonth int     // Mois de vente estimé (1-based)
	BridgeLoanRepayPct  float64 // Part affectée au remboursement anticipé (%)
	BridgeLoanRepayLine int     // Index de la ligne de prêt prioritaire pour le remboursement
}

// EnergyComparisonYear holds the energy cost comparison data for one year.
type EnergyComparisonYear struct {
	Year         int
	CumulGas1    float64 // Cumul gaz bien 1
	CumulElec1   float64 // Cumul électricité bien 1
	CumulOther1  float64 // Cumul autres coûts bien 1
	CumulTotal1  float64 // Cumul total bien 1
	CumulGas2    float64 // Cumul gaz bien 2
	CumulElec2   float64 // Cumul électricité bien 2
	CumulOther2  float64 // Cumul autres coûts bien 2
	CumulTotal2  float64 // Cumul total bien 2
	CumulSavings float64 // Total1 - Total2 (positif = bien 2 moins cher)
	CumulGas3    float64 // Cumul gaz bien 3
	CumulElec3   float64 // Cumul électricité bien 3
	CumulOther3  float64 // Cumul autres coûts bien 3
	CumulTotal3  float64 // Cumul total bien 3
	// kWh tracking
	CumulGasKWh1  float64 // Cumul kWh gaz bien 1
	CumulElecKWh1 float64 // Cumul kWh électricité bien 1
	CumulGasKWh2  float64 // Cumul kWh gaz bien 2
	CumulElecKWh2 float64 // Cumul kWh électricité bien 2
	CumulGasKWh3  float64 // Cumul kWh gaz bien 3
	CumulElecKWh3 float64 // Cumul kWh électricité bien 3
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
	BrokerFees       float64 // Frais de courtage
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
	PropertySale                PropertySale                     // Résultat de la vente du bien actuel
	CurrentPropertyProjection   []CurrentPropertyMonthProjection // Projection bien actuel (E1/E2)
	AidEligibility              AidEligibility                  // Éligibilité aux aides (PTZ, PAL, BRS)
	LoanLineResults       []NewLoanLineResult // Détail des résultats par ligne de crédit
	EquivalentRent        float64             // Loyer équivalent mensuel (coûts irrécupérables récurrents / mois)
	MonthlySchedule       []MonthlySchedule   // Planning mensuel lissé avec répartition par prêt
	CurrentLoanSchedule     []MonthlySchedule         // Planning mensuel des prêts en cours (bien actuel)
	CurrentBorrowerPayments []CurrentBorrowerPayment  // Versements cumulés par emprunteur (bien actuel)
	EnergyComparisonData    []EnergyComparisonYear    // Données comparaison coûts énergétiques
	BridgeLoan              BridgeLoanResult          // Résultat du prêt relais
}

// BridgeLoanResult holds the computed results for a bridge loan (prêt relais).
type BridgeLoanResult struct {
	Enabled           bool    // Prêt relais activé
	Amount            float64 // Montant du prêt relais (quotité × prix vente)
	NetAmount         float64 // Montant net disponible (Amount − CRD)
	Rate              float64 // Taux appliqué
	Duration          int     // Durée en mois
	Franchise         string  // Type de franchise
	MonthlyPayment    float64 // Mensualité pendant la période relais (0 si franchise totale)
	TotalInterest     float64 // Coût total des intérêts
	TotalInsurance    float64 // Coût total assurance
	TotalCost         float64 // Coût total du prêt relais
	CapitalizedAmount float64 // Montant à rembourser à la vente (capital + intérêts capitalisés si franchise totale)
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

// CurrentPropertyMonthProjection holds one month's projection for the current property E1/E2 split.
type CurrentPropertyMonthProjection struct {
	Month         int       // Mois depuis aujourd'hui (1, 2, 3, ...)
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

// CurrentBorrowerPayment holds cumulative payments per borrower at each month for current property.
type CurrentBorrowerPayment struct {
	Month         int     `json:"month"`
	CumulPayment1 float64 `json:"cumulPayment1"` // Versements cumulés E1 (mensualités prêt)
	CumulPayment2 float64 `json:"cumulPayment2"` // Versements cumulés E2 (contributions virtuelles)
}

// IrrecoverableDetail holds the breakdown of irrecoverable costs.
type IrrecoverableDetail struct {
	NotaryFees       float64 // Frais de notaire
	AgencyFees       float64 // Frais d'agence
	BankFees         float64 // Frais de dossier
	BrokerFees       float64 // Frais de courtage
	TotalInterest    float64 // Intérêts cumulés
	TotalInsurance   float64 // Assurance cumulée
	TotalCondoFees   float64 // Charges copro cumulées
	TotalPropertyTax float64 // Taxe foncière cumulée
	TotalMaintenance float64 // Frais d'entretien cumulés
	Total            float64 // Total des coûts irrécupérables
}

// LoanPaymentDetail holds cumulative payment info for a single loan line.
type LoanPaymentDetail struct {
	Label     string  // Libellé du prêt
	Amount    float64 // Montant cumulé des versements (total)
	Principal float64 // Capital amorti cumulé
	Interest  float64 // Intérêts cumulés
	Insurance float64 // Assurance cumulée
}

// PropertySale holds the result of selling the current property.
type PropertySale struct {
	SalePrice          float64             // Prix de vente
	LoanBalance        float64             // Capital restant dû (total)
	Penalty            float64             // IRA (total)
	NetProceeds        float64             // Produit net de vente
	Proceeds1          float64             // Part emprunteur 1
	Proceeds2          float64             // Part emprunteur 2
	LoanLines          []LoanLine          // Détail des lignes de prêt
	ApportE1Initial    float64             // Apport initial E1 (down payment)
	ApportE1Loans      float64             // Versements prêts cumulés E1
	ApportE1LoansDetail []LoanPaymentDetail // Détail versements par ligne de prêt
	ApportE1Total      float64             // Total contribution E1 (apport + prêts - remboursements E2)
	ApportE2Initial    float64             // Apport initial E2
	ApportE2Monthly    float64             // Mensualités cumulées E2
	ApportE2Total      float64             // Total contribution E2
	TotalApports       float64             // Total des apports (E1 + E2)
	ContributionPctE1  float64             // Pourcentage contribution E1
	ContributionPctE2  float64             // Pourcentage contribution E2
	MonthsElapsed      int                 // Mois écoulés depuis début du prêt
	Profit             float64             // Bénéfice (ProduitNet - TotalApports)
	ProfitShareE2      float64             // Part du bénéfice pour E2
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
	CumulInterest    float64 // Cumul des intérêts
	Insurance        float64 // Assurance
	RemainingBalance float64 // Capital restant dû
}

// PaymentTier represents a payment period with a fixed monthly payment.
type PaymentTier struct {
	StartMonth     int     `json:"startMonth"`     // Mois de début (1 = premier mois du prêt)
	EndMonth       int     `json:"endMonth"`       // Mois de fin (inclus)
	MonthlyPayment float64 `json:"monthlyPayment"` // Mensualité pour cette période
}

// LoanLine represents a single loan line for IRA calculation.
type LoanLine struct {
	Label            string        `json:"label"`            // Libellé du prêt (ex: "Prêt principal", "PTZ")
	OriginalAmount   float64       `json:"originalAmount"`   // Montant emprunté initial
	Balance          float64       `json:"balance"`          // Capital restant dû
	Rate             float64       `json:"rate"`             // Taux d'intérêt (%)
	IRA              float64       `json:"ira"`              // IRA saisi par l'utilisateur
	StartYear        int           `json:"startYear"`        // Année de début du prêt
	StartMonth       int           `json:"startMonth"`       // Mois de début (1-12)
	DurationYears    int           `json:"durationYears"`    // Durée totale en années
	InsuranceRate    float64       `json:"insuranceRate"`    // Taux assurance annuel (%)
	InsuranceMonthly float64       `json:"insuranceMonthly"` // Mensualité assurance (€) - alternatif au taux
	DeferralMonths   int           `json:"deferralMonths"`   // Différé en mois (paiement intérêts seuls)
	DeferralRate     float64       `json:"deferralRate"`     // Taux intérêts intercalaires (%)
	Tiers            []PaymentTier `json:"tiers"`            // Paliers de paiement
}

// NewLoanLine represents a loan line for the new mortgage.
type NewLoanLine struct {
	Label          string        `json:"label"`          // Libellé du prêt (ex: "Prêt principal", "PTZ", "PAL")
	Amount         float64       `json:"amount"`         // Montant emprunté (€)
	Rate           float64       `json:"rate"`           // Taux d'intérêt annuel (%)
	DurationYears  int           `json:"durationYears"`  // Durée en années
	InsuranceRate  float64       `json:"insuranceRate"`  // Taux assurance annuel (%)
	DeferralMonths int           `json:"deferralMonths"` // Différé en mois (paiement intérêts seuls)
	DeferralRate   float64       `json:"deferralRate"`   // Taux intérêts intercalaires (%)
	Tiers          []PaymentTier `json:"tiers"`          // Paliers de paiement
}

// LoanMonthPayment holds the payment details for a single loan in a given month.
type LoanMonthPayment struct {
	Label     string  // Libellé du prêt
	Principal float64 // Capital remboursé
	Interest  float64 // Intérêts
	Insurance float64 // Assurance
	Total     float64 // Total pour ce prêt
}

// MonthlySchedule holds the payment breakdown for all loans in a given month.
type MonthlySchedule struct {
	Month       int                // Numéro du mois (1-based)
	Payments    []LoanMonthPayment // Paiement par ligne de crédit
	TotalAmount float64            // Somme des paiements
}

// NewLoanLineResult holds computed results for a single loan line.
type NewLoanLineResult struct {
	Label            string  // Libellé du prêt
	Amount           float64 // Montant emprunté
	Rate             float64 // Taux d'intérêt
	DurationYears    int     // Durée
	DeferralMonths   int     // Différé en mois
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
