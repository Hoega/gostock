# Simulation de crédit immobilier

*Générée le {{now}}*

## Projet

- **Prix du bien** : {{formatMoney .Input.PropertyPrice}} €
- Travaux : {{formatMoney .Input.RenovationCost}} €
- Frais de notaire : {{formatMoney .Result.NotaryFees}} €
- Frais d'agence : {{formatMoney .Result.AgencyFees}} €
- Frais de dossier : {{formatMoney .Result.BankFees}} €
- Frais de garantie : {{formatMoney .Result.GuaranteeFees}} €
- Frais de courtage : {{formatMoney .Result.BrokerFees}} €
- **Coût total du projet** : {{formatMoney .Result.TotalProjectCost}} €

## Apports

- Emprunteur 1 : {{formatMoney .Input.DownPayment1}} €
- Emprunteur 2 : {{formatMoney .Input.DownPayment2}} €
{{if gt .Input.CurrentSalePrice 0.0}}- Prix de vente bien actuel : {{formatMoney .Input.CurrentSalePrice}} €
- Capital restant dû : {{formatMoney .Input.CurrentLoanBalance}} €
- IRA : {{formatMoney .Input.EarlyRepaymentPenalty}} €
- **Produit net vente** : {{formatMoney .Result.PropertySale.NetProceeds}} €
{{end}}

## Offre banque

### Lignes de crédit

| Libellé | Montant | Taux | Durée | Mensualité hors ass. | Mensualité totale |
|---|---:|---:|---:|---:|---:|
{{range .Result.LoanLineResults}}| {{.Label}} | {{formatMoney .Amount}} € | {{printf "%.2f" .Rate}} % | {{.DurationYears}} ans | {{formatMoney .MonthlyPayment}} € | {{formatMoney .MonthlyTotal}} € |
{{end}}
{{if .Result.BridgeLoan.Enabled}}
### Prêt relais

- Montant : {{formatMoney .Result.BridgeLoan.Amount}} € (quotité {{printf "%.0f" .Input.BridgeLoanQuotity}} %)
- Net disponible : {{formatMoney .Result.BridgeLoan.NetAmount}} €
- Taux : {{printf "%.2f" .Result.BridgeLoan.Rate}} %
- Durée : {{.Result.BridgeLoan.Duration}} mois
- Franchise : {{.Result.BridgeLoan.Franchise}}
- Mensualité pendant relais : {{formatMoney .Result.BridgeLoan.MonthlyPayment}} €
- **À rembourser à la vente** : {{formatMoney .Result.BridgeLoan.CapitalizedAmount}} €
- Coût total du relais : {{formatMoney .Result.BridgeLoan.TotalCost}} €
{{end}}

## Synthèse financière

- Mensualité hors assurance : {{formatMoney .Result.MonthlyPayment}} €
- Mensualité assurance : {{formatMoney .Result.MonthlyInsurance}} €
- **Mensualité totale** : {{formatMoney .Result.MonthlyTotal}} €
- Revenus mensuels nets : {{formatMoney .Result.IncomeMonthly}} €
- Taux d'effort : {{printf "%.2f" .Result.EffortRate}} %
- Mensualité max HCSF (35 %) : {{formatMoney .Result.MaxMonthlyPayment}} €
- Capacité d'emprunt max : {{formatMoney .Result.MaxLoanAmount}} €
- Total intérêts : {{formatMoney .Result.TotalInterest}} €
- Total assurance : {{formatMoney .Result.TotalInsurance}} €
- **Coût total du crédit** : {{formatMoney .Result.TotalLoanCost}} €

## Aides

{{with .Result.AidEligibility}}- PTZ : {{if .PTZEligible}}éligible — plafond {{formatMoney .PTZMaxAmount}} € (ressources ≤ {{formatMoney .PTZIncomeCeiling}} €){{else}}non éligible{{end}}
- PAL : {{if .PALEligible}}éligible — max {{formatMoney .PALMaxAmount}} €{{else}}non éligible{{end}}
- BRS : {{if .BRSEligible}}éligible{{else}}non éligible{{end}}
{{end}}

## Tableau d'amortissement

{{if .Result.MonthlySchedule}}| Mois | Date | Mensualité | Capital | Intérêts | Assurance |{{if .Result.BridgeLoan.Enabled}} Relais |{{end}} Capital restant |
|---:|:---|---:|---:|---:|---:|{{if .Result.BridgeLoan.Enabled}}---:|{{end}}---:|
{{range .Result.MonthlySchedule}}| {{.Month}} | {{.Date}} | {{formatMoney .TotalAmount}} € | {{formatMoney .TotalPrincipal}} € | {{formatMoney .TotalInterest}} € | {{formatMoney .TotalInsurance}} € |{{if $.Result.BridgeLoan.Enabled}} {{if .BridgeActive}}{{formatMoney .BridgePayment}} €{{else}}—{{end}} |{{end}} {{formatMoney .RemainingBalance}} € |
{{end}}{{else}}| Mois | Date | Mensualité | Capital | Intérêts | Assurance | Capital restant |
|---:|:---|---:|---:|---:|---:|---:|
{{range .Result.Amortization}}| {{.Month}} | {{.Date}} | {{formatMoney .Payment}} € | {{formatMoney .Principal}} € | {{formatMoney .Interest}} € | {{formatMoney .Insurance}} € | {{formatMoney .RemainingBalance}} € |
{{end}}{{end}}
