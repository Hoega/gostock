# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build
go build -o gostock ./cmd/gostock/

# Run (default port 8080)
./gostock

# Run with custom port
./gostock -port 9999
PORT=3000 ./gostock
```

## Architecture

GoStock is a French mortgage/credit simulator web application built with Go, Chi router, HTMX, and Chart.js.

### Package Structure

- **cmd/gostock/main.go** - Entry point, flag parsing, graceful shutdown with signal handling
- **internal/server/** - HTTP server setup, Chi router configuration, template loading
- **internal/handler/** - HTTP request handlers for form display and calculation
- **internal/calculator/** - Pure calculation logic (amortization, resale profitability, rent vs buy)
- **internal/model/** - Data structures (`CreditInput`, `CreditResult`, amortization rows)
- **web/templates/** - Go HTML templates with layout and partials
- **web/static/** - Static assets

### Request Flow

1. `GET /credit` renders the main form (`credit.html`)
2. `POST /credit/calculate` parses form inputs, runs `calculator.Calculate()`, returns 4 HTML partials via HTMX
3. Partials: results summary, amortization table, charts, rent-vs-buy comparison

### Template System

Templates use `html/template` with custom functions:
- `formatMoney` - French currency formatting (space thousand separator)
- `seq` - Integer sequence generation
- `toJSON` - Go value to JSON conversion

### Key Design Patterns

- HTMX-driven: Form submissions return HTML partials, no JSON APIs
- French localization throughout (month names, labels, formatting)
- Calculator package is pure functions with no side effects
- All monetary calculations round to 2 decimal places
