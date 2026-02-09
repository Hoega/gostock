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
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
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
	}).ParseGlob("web/templates/*.html"))
	template.Must(tmpl.ParseGlob("web/templates/partials/*.html"))

	creditHandler := handler.NewCreditHandler(tmpl, store)

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
