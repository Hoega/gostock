package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Hoega/gostock/internal/handler"
	"github.com/Hoega/gostock/internal/persistence"
)

func New(port int, store persistence.Store) *http.Server {
	funcMap := template.FuncMap{
		"formatMoney": func(v float64) string {
			// French-style money formatting with space as thousand separator
			s := fmt.Sprintf("%.2f", v)
			parts := [2]string{s[:len(s)-3], s[len(s)-3:]} // integer part, decimal part
			intPart := parts[0]
			n := len(intPart)
			var result []byte
			for i, c := range intPart {
				if i > 0 && (n-i)%3 == 0 && c != '-' {
					result = append(result, ' ')
				}
				result = append(result, intPart[i])
			}
			return string(result) + parts[1]
		},
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
		"toJSON": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"sub": func(a, b any) float64 {
			return toFloat(a) - toFloat(b)
		},
		"subInt": func(a, b int) int {
			return a - b
		},
		"add": func(a, b any) float64 {
			return toFloat(a) + toFloat(b)
		},
		"mul": func(a, b any) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			return af * bf
		},
		"div": func(a, b any) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			if bf == 0 {
				return 0
			}
			return af / bf
		},
		"percentInRange": func(current, low, high float64) float64 {
			if high == low {
				return 50
			}
			pct := (current - low) / (high - low) * 100
			if pct < 0 {
				return 0
			}
			if pct > 100 {
				return 100
			}
			return pct
		},
	}

	// Parse base templates (layout + partials) then clone per page
	// to avoid "content" template name collisions between pages
	baseTmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/partials/*.html"))
	template.Must(baseTmpl.ParseFiles("web/templates/layout.html"))

	creditTmpl := template.Must(template.Must(baseTmpl.Clone()).ParseFiles("web/templates/credit.html"))
	portfolioTmpl := template.Must(template.Must(baseTmpl.Clone()).ParseFiles("web/templates/portfolio.html"))
	dashboardTmpl := template.Must(template.Must(baseTmpl.Clone()).ParseFiles("web/templates/dashboard.html"))
	taxTmpl := template.Must(template.Must(baseTmpl.Clone()).ParseFiles("web/templates/tax.html"))

	creditHandler := handler.NewCreditHandler(creditTmpl, store)
	portfolioHandler := handler.NewPortfolioHandler(portfolioTmpl, store)
	dashboardHandler := handler.NewDashboardHandler(dashboardTmpl, store)
	taxHandler := handler.NewTaxHandler(taxTmpl, store)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Redirect root to credit simulator
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/credit", http.StatusFound)
	})

	// Credit simulator routes
	r.Get("/credit", creditHandler.ShowForm)
	r.Post("/credit/calculate", creditHandler.Calculate)

	// Dashboard route
	r.Get("/dashboard", dashboardHandler.ShowDashboard)

	// Portfolio routes - Stocks
	r.Get("/portfolio", portfolioHandler.ShowPortfolio)
	r.Get("/portfolio/quote", portfolioHandler.LookupQuote)
	r.Get("/portfolio/history", portfolioHandler.StockHistory)
	r.Get("/portfolio/history/total", portfolioHandler.TotalHistory)
	r.Post("/portfolio/positions", portfolioHandler.AddPosition)
	r.Put("/portfolio/positions/{id}", portfolioHandler.UpdatePosition)
	r.Delete("/portfolio/positions/{id}", portfolioHandler.DeletePosition)

	// Portfolio routes - Crypto
	r.Get("/portfolio/crypto/quote", portfolioHandler.LookupCryptoQuote)
	r.Get("/portfolio/crypto/history", portfolioHandler.CryptoHistory)
	r.Post("/portfolio/crypto/positions", portfolioHandler.AddCryptoPosition)
	r.Put("/portfolio/crypto/positions/{id}", portfolioHandler.UpdateCryptoPosition)
	r.Delete("/portfolio/crypto/positions/{id}", portfolioHandler.DeleteCryptoPosition)

	// Tax routes
	r.Get("/tax", taxHandler.ShowTax)
	r.Post("/tax/stocks/sales", taxHandler.AddStockSale)
	r.Put("/tax/stocks/sales/{id}", taxHandler.UpdateStockSale)
	r.Delete("/tax/stocks/sales/{id}", taxHandler.DeleteStockSale)
	r.Post("/tax/crypto/sales", taxHandler.AddCryptoSale)
	r.Put("/tax/crypto/sales/{id}", taxHandler.UpdateCryptoSale)
	r.Delete("/tax/crypto/sales/{id}", taxHandler.DeleteCryptoSale)
	// Stock purchases for PRU calculation
	r.Get("/tax/purchases/pru", taxHandler.GetPRUForISIN)
	r.Post("/tax/purchases", taxHandler.AddStockPurchase)
	r.Put("/tax/purchases/{id}", taxHandler.UpdateStockPurchase)
	r.Delete("/tax/purchases/{id}", taxHandler.DeleteStockPurchase)
	r.Post("/tax/purchases/{id}/reset", taxHandler.ResetStockPurchase)

	// Static files
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func Start(port int, store persistence.Store) *http.Server {
	srv := New(port, store)
	log.Printf("Server starting on http://localhost:%d", port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	return srv
}

// toFloat converts any numeric type to float64
func toFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}
