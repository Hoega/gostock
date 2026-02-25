package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/Hoega/gostock/internal/calculator"
	"github.com/Hoega/gostock/internal/model"
	"github.com/Hoega/gostock/internal/persistence"
)

// BudgetHandler handles the budget Sankey diagram page.
type BudgetHandler struct {
	templates *template.Template
	store     persistence.Store
}

func NewBudgetHandler(templates *template.Template, store persistence.Store) *BudgetHandler {
	return &BudgetHandler{templates: templates, store: store}
}

// SubscriptionItem represents a single subscription entry.
type SubscriptionItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// BudgetData is the template data for the budget page.
type BudgetData struct {
	Inputs             *persistence.BudgetInputs
	SubscriptionItems  []SubscriptionItem
	LifestyleItems     []SubscriptionItem
	OtherExpensesItems []SubscriptionItem
	HasMortgage        bool
	MortgageTotal      float64
	MortgageCapital    float64
	MortgageInterest   float64
	MortgageInsurance  float64
}

// ShowBudget handles GET /budget.
func (h *BudgetHandler) ShowBudget(w http.ResponseWriter, r *http.Request) {
	inputs, err := h.store.LoadBudgetInputs()
	if err != nil {
		log.Printf("Failed to load budget inputs: %v", err)
		inputs = persistence.DefaultBudgetInputs()
	}

	data := h.buildBudgetData(inputs)

	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.ExecuteTemplate(w, "budget-content.html", data); err != nil {
			log.Printf("Budget content template error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.ExecuteTemplate(w, "budget.html", data); err != nil {
		log.Printf("Budget template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SaveBudget handles POST /budget/save.
func (h *BudgetHandler) SaveBudget(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	inputs := &persistence.BudgetInputs{
		GrossSalary:   parseBudgetFloat(r.FormValue("gross_salary")),
		NetSalary:     parseBudgetFloat(r.FormValue("net_salary")),
		Dividends:     parseBudgetFloat(r.FormValue("dividends")),
		RentalIncome:  parseBudgetFloat(r.FormValue("rental_income")),
		OtherIncome:   parseBudgetFloat(r.FormValue("other_income")),
		IncomeTax:     parseBudgetFloat(r.FormValue("income_tax")),
		Housing:       parseBudgetFloat(r.FormValue("housing")),
		Lifestyle:     parseBudgetFloat(r.FormValue("lifestyle")),
		Transport:     parseBudgetFloat(r.FormValue("transport")),
		Insurance:        parseBudgetFloat(r.FormValue("insurance")),
		Subscriptions:    parseBudgetFloat(r.FormValue("subscriptions")),
		Childcare:        parseBudgetFloat(r.FormValue("childcare")),
		MealVouchers:     parseBudgetFloat(r.FormValue("meal_vouchers")),
		SubscriptionsJSON: r.FormValue("subscriptions_json"),
		LifestyleJSON:     r.FormValue("lifestyle_json"),
		OtherExpenses:    parseBudgetFloat(r.FormValue("other_expenses")),
		OtherExpensesJSON: r.FormValue("other_expenses_json"),
		PEA:           parseBudgetFloat(r.FormValue("pea")),
		AssuranceVie:  parseBudgetFloat(r.FormValue("assurance_vie")),
		PER:           parseBudgetFloat(r.FormValue("per")),
		LivretA:       parseBudgetFloat(r.FormValue("livret_a")),
		CryptoSavings: parseBudgetFloat(r.FormValue("crypto_savings")),
		OtherSavings:  parseBudgetFloat(r.FormValue("other_savings")),
	}

	// Validate JSON fields — default to empty array
	if inputs.SubscriptionsJSON == "" {
		inputs.SubscriptionsJSON = "[]"
	}
	if inputs.LifestyleJSON == "" {
		inputs.LifestyleJSON = "[]"
	}
	if inputs.OtherExpensesJSON == "" {
		inputs.OtherExpensesJSON = "[]"
	}

	if err := h.store.SaveBudgetInputs(inputs); err != nil {
		log.Printf("Failed to save budget inputs: %v", err)
		http.Error(w, "Erreur de sauvegarde", http.StatusInternalServerError)
		return
	}

	data := h.buildBudgetData(inputs)
	if err := h.templates.ExecuteTemplate(w, "budget-content.html", data); err != nil {
		log.Printf("Budget content template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SankeyNode represents a node in the Sankey diagram.
type SankeyNode struct {
	Name string `json:"name"`
}

// SankeyLink represents a link between nodes.
type SankeyLink struct {
	Source int     `json:"source"`
	Target int     `json:"target"`
	Value  float64 `json:"value"`
}

// SankeyData is the JSON response for the Sankey chart.
type SankeyData struct {
	Nodes []SankeyNode `json:"nodes"`
	Links []SankeyLink `json:"links"`
}

// GetBudgetSankey handles GET /budget/sankey — returns JSON for D3.
func (h *BudgetHandler) GetBudgetSankey(w http.ResponseWriter, r *http.Request) {
	inputs, err := h.store.LoadBudgetInputs()
	if err != nil {
		log.Printf("Failed to load budget inputs: %v", err)
		inputs = persistence.DefaultBudgetInputs()
	}

	data := h.buildSankeyData(inputs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// parseSubscriptionItems parses the JSON subscription list.
func parseSubscriptionItems(jsonStr string) []SubscriptionItem {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	var items []SubscriptionItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		log.Printf("Failed to parse subscriptions JSON: %v", err)
		return nil
	}
	return items
}

// buildBudgetData builds the template data, including mortgage detection.
func (h *BudgetHandler) buildBudgetData(inputs *persistence.BudgetInputs) BudgetData {
	data := BudgetData{
		Inputs:             inputs,
		SubscriptionItems:  parseSubscriptionItems(inputs.SubscriptionsJSON),
		LifestyleItems:     parseSubscriptionItems(inputs.LifestyleJSON),
		OtherExpensesItems: parseSubscriptionItems(inputs.OtherExpensesJSON),
	}

	// Try to load mortgage data from credit simulator
	capital, interest, insurance := h.getMortgageSplit()
	if capital > 0 || interest > 0 || insurance > 0 {
		data.HasMortgage = true
		data.MortgageCapital = capital
		data.MortgageInterest = interest
		data.MortgageInsurance = insurance
		data.MortgageTotal = capital + interest + insurance
	}

	return data
}

// getMortgageSplit loads credit simulator data and computes the current monthly split.
func (h *BudgetHandler) getMortgageSplit() (capital, interest, insurance float64) {
	formInputs, err := h.store.Load()
	if err != nil {
		return 0, 0, 0
	}

	if formInputs.CurrentLoanLines == "" || formInputs.CurrentLoanLines == "[]" {
		return 0, 0, 0
	}

	var loanLines []model.LoanLine
	if err := json.Unmarshal([]byte(formInputs.CurrentLoanLines), &loanLines); err != nil {
		log.Printf("Failed to parse loan lines: %v", err)
		return 0, 0, 0
	}

	now := time.Now()
	atYear := now.Year()
	atMonth := int(now.Month())

	for _, line := range loanLines {
		remaining, _, monthly, monthlyInsurance := calculator.ComputeLoanRemainingBalance(line, atYear, atMonth)
		if remaining <= 0 || monthly <= 0 {
			continue
		}
		// monthly includes insurance already, so: principal+interest = monthly - insurance
		// interest for current month = remaining_balance * monthly_rate
		monthlyRate := line.Rate / 100 / 12
		monthInterest := remaining * monthlyRate
		monthPrincipal := monthly - monthlyInsurance - monthInterest
		if monthPrincipal < 0 {
			monthPrincipal = 0
		}

		capital += math.Round(monthPrincipal*100) / 100
		interest += math.Round(monthInterest*100) / 100
		insurance += math.Round(monthlyInsurance*100) / 100
	}

	return capital, interest, insurance
}

// buildSankeyData builds the nodes and links for the Sankey diagram.
func (h *BudgetHandler) buildSankeyData(inputs *persistence.BudgetInputs) SankeyData {
	var nodes []SankeyNode
	var links []SankeyLink

	nodeIndex := map[string]int{}
	addNode := func(name string) int {
		if idx, ok := nodeIndex[name]; ok {
			return idx
		}
		idx := len(nodes)
		nodes = append(nodes, SankeyNode{Name: name})
		nodeIndex[name] = idx
		return idx
	}
	addLink := func(source, target int, value float64) {
		if value > 0 {
			links = append(links, SankeyLink{Source: source, Target: target, Value: math.Round(value*100) / 100})
		}
	}

	// === Income sources (left) ===
	idxGrossSalary := addNode("Salaire brut")
	idxDividends := addNode("Dividendes")
	idxRentalIncome := addNode("Revenus locatifs")
	idxOtherIncome := addNode("Autres revenus")

	// === Central hubs ===
	idxNetSalary := addNode("Salaire net")
	idxRevenus := addNode("Revenus nets")    // before tax
	idxDisponible := addNode("Revenu disponible") // after tax

	// === Deductions (middle) ===
	idxCotisations := addNode("Cotisations sociales")
	idxIncomeTax := addNode("Impôt sur le revenu")
	idxHousing := addNode("Logement")
	idxLifestyle := addNode("Dépenses courantes")
	idxTransport := addNode("Transport")
	idxInsurance := addNode("Assurances")
	idxChildcare := addNode("Mode de garde")
	idxMealVouchers := addNode("Tickets restaurant")
	idxSubscriptions := addNode("Abonnements")
	idxOtherExpenses := addNode("Autres dépenses")

	// === Savings (right) ===
	idxPEA := addNode("PEA")
	idxAssuranceVie := addNode("Assurance Vie")
	idxPER := addNode("PER")
	idxLivretA := addNode("Livret A")
	idxCrypto := addNode("Crypto")
	idxOtherSavings := addNode("Autres épargnes")

	// Mortgage split nodes (only if mortgage data exists)
	mortgageCapital, mortgageInterest, mortgageInsurance := h.getMortgageSplit()
	hasMortgage := mortgageCapital > 0 || mortgageInterest > 0 || mortgageInsurance > 0

	var idxMortgage, idxMortgageCapital, idxMortgageInterest, idxMortgageInsurance int
	if hasMortgage {
		idxMortgage = addNode("Crédit immobilier")
		idxMortgageCapital = addNode("Capital amorti")
		idxMortgageInterest = addNode("Intérêts")
		idxMortgageInsurance = addNode("Assurance emprunt")
	}

	// === Links ===

	// Gross salary → Cotisations + Net salary
	cotisations := inputs.GrossSalary - inputs.NetSalary
	if cotisations > 0 {
		addLink(idxGrossSalary, idxCotisations, cotisations)
	}
	addLink(idxGrossSalary, idxNetSalary, inputs.NetSalary)

	// Net salary → Revenus nets
	addLink(idxNetSalary, idxRevenus, inputs.NetSalary)

	// Other income sources → Revenus nets
	addLink(idxDividends, idxRevenus, inputs.Dividends)
	addLink(idxRentalIncome, idxRevenus, inputs.RentalIncome)
	addLink(idxOtherIncome, idxRevenus, inputs.OtherIncome)

	// Revenus nets → Impôt + Revenu disponible (after tax)
	addLink(idxRevenus, idxIncomeTax, inputs.IncomeTax)
	totalRevenus := inputs.NetSalary + inputs.Dividends + inputs.RentalIncome + inputs.OtherIncome
	disponible := totalRevenus - inputs.IncomeTax
	if disponible > 0 {
		addLink(idxRevenus, idxDisponible, disponible)
	}
	// Lifestyle: individual sub-nodes if items exist, otherwise single node
	lifestyleItems := parseSubscriptionItems(inputs.LifestyleJSON)
	if len(lifestyleItems) > 0 {
		var lifestyleTotal float64
		for _, item := range lifestyleItems {
			lifestyleTotal += item.Amount
		}
		if lifestyleTotal > 0 {
			addLink(idxDisponible, idxLifestyle, lifestyleTotal)
			for _, item := range lifestyleItems {
				if item.Amount > 0 && item.Name != "" {
					idx := addNode(item.Name)
					addLink(idxLifestyle, idx, item.Amount)
				}
			}
		}
	} else if inputs.Lifestyle > 0 {
		addLink(idxDisponible, idxLifestyle, inputs.Lifestyle)
	}

	addLink(idxDisponible, idxTransport, inputs.Transport)
	addLink(idxDisponible, idxInsurance, inputs.Insurance)
	addLink(idxDisponible, idxChildcare, inputs.Childcare)
	addLink(idxDisponible, idxMealVouchers, inputs.MealVouchers)

	// Other expenses: individual sub-nodes if items exist, otherwise single node
	otherExpItems := parseSubscriptionItems(inputs.OtherExpensesJSON)
	if len(otherExpItems) > 0 {
		var otherExpTotal float64
		for _, item := range otherExpItems {
			otherExpTotal += item.Amount
		}
		if otherExpTotal > 0 {
			addLink(idxDisponible, idxOtherExpenses, otherExpTotal)
			for _, item := range otherExpItems {
				if item.Amount > 0 && item.Name != "" {
					idx := addNode(item.Name)
					addLink(idxOtherExpenses, idx, item.Amount)
				}
			}
		}
	} else if inputs.OtherExpenses > 0 {
		addLink(idxDisponible, idxOtherExpenses, inputs.OtherExpenses)
	}

	// Subscriptions: individual sub-nodes if items exist, otherwise single node
	subItems := parseSubscriptionItems(inputs.SubscriptionsJSON)
	if len(subItems) > 0 {
		var subTotal float64
		for _, item := range subItems {
			subTotal += item.Amount
		}
		if subTotal > 0 {
			addLink(idxDisponible, idxSubscriptions, subTotal)
			for _, item := range subItems {
				if item.Amount > 0 && item.Name != "" {
					idx := addNode(item.Name)
					addLink(idxSubscriptions, idx, item.Amount)
				}
			}
		}
	} else if inputs.Subscriptions > 0 {
		// Fallback: legacy scalar field
		addLink(idxDisponible, idxSubscriptions, inputs.Subscriptions)
	}

	// Housing: mortgage split or rent
	if hasMortgage {
		mortgageTotal := mortgageCapital + mortgageInterest + mortgageInsurance
		addLink(idxDisponible, idxMortgage, mortgageTotal)
		addLink(idxMortgage, idxMortgageCapital, mortgageCapital)
		addLink(idxMortgage, idxMortgageInterest, mortgageInterest)
		addLink(idxMortgage, idxMortgageInsurance, mortgageInsurance)
	} else {
		addLink(idxDisponible, idxHousing, inputs.Housing)
	}

	// Revenu disponible → Savings
	addLink(idxDisponible, idxPEA, inputs.PEA)
	addLink(idxDisponible, idxAssuranceVie, inputs.AssuranceVie)
	addLink(idxDisponible, idxPER, inputs.PER)
	addLink(idxDisponible, idxLivretA, inputs.LivretA)
	addLink(idxDisponible, idxCrypto, inputs.CryptoSavings)
	addLink(idxDisponible, idxOtherSavings, inputs.OtherSavings)

	// Filter out nodes that have no links
	return filterSankeyData(nodes, links)
}

// filterSankeyData removes orphan nodes and re-indexes links.
func filterSankeyData(nodes []SankeyNode, links []SankeyLink) SankeyData {
	// Find which nodes are actually referenced
	used := map[int]bool{}
	for _, l := range links {
		used[l.Source] = true
		used[l.Target] = true
	}

	// Build new node list and mapping
	newIdx := map[int]int{}
	var filtered []SankeyNode
	for i, n := range nodes {
		if used[i] {
			newIdx[i] = len(filtered)
			filtered = append(filtered, n)
		}
	}

	// Re-index links
	var newLinks []SankeyLink
	for _, l := range links {
		newLinks = append(newLinks, SankeyLink{
			Source: newIdx[l.Source],
			Target: newIdx[l.Target],
			Value:  l.Value,
		})
	}

	return SankeyData{Nodes: filtered, Links: newLinks}
}

func parseBudgetFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
