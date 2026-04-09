# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

The project uses [Task](https://taskfile.dev/) as its task runner. Prefer `task` commands over raw `go` commands.

```bash
# Build
task build              # or: go build -o gostock ./cmd/gostock/

# Run (default port 8080, must run from project root — templates use relative paths)
task run                # builds then runs
task dev                # builds, runs, and auto-restarts on file changes

# Run with custom port
./gostock -port 9999
PORT=3000 ./gostock

# Code quality
task fmt                # gofmt -w .
task vet                # go vet ./...
task check              # fmt + vet + build (pre-commit quality check)

# Database
task db:path            # print SQLite DB path
task db:backup          # backup DB with timestamp

# Cleanup
task stop               # kill running server
task clean              # remove built binary
```

There are no tests or linter configurations in this project. Always verify changes compile with `task build` (or `go build`).

## Architecture

GoStock is a French personal finance web application built with Go 1.25.5, Chi router, HTMX, and Chart.js. Uses pure-Go SQLite via `modernc.org/sqlite` (no CGO required) with `sqlx` for query binding. It combines:
- **Mortgage/credit simulation** with French-specific calculations (PTZ, HCSF rules)
- **Portfolio tracking** for stocks, crypto, and cash
- **Tax reporting** for French tax forms (2042-C, 2086)
- **Dashboard** with global wealth overview

### Package Structure

- **cmd/gostock/main.go** - Entry point, flag parsing, graceful shutdown
- **internal/server/** - HTTP server, Chi router, template loading with custom FuncMap
- **internal/handler/** - HTTP handlers: `credit`, `compare`, `portfolio`, `dashboard`, `tax`, `budget`
- **internal/calculator/** - Pure calculation logic (amortization, performance, tax)
- **internal/model/** - Domain structs with computed fields and summary logic
- **internal/persistence/** - SQLite storage with `Store` interface; has its own struct types mirroring `model/` (handlers convert between them)
- **internal/quote/** - Market data: Yahoo Finance (stocks), CoinGecko (crypto), and INSEE (French inflation data)
- **web/templates/** - Go HTML templates with layout and partials
- **web/static/** - Static assets (JS, CSS)

### Dual Model Types

`persistence/store.go` defines DB-layer structs (e.g. `persistence.StockPosition`) and `model/portfolio.go` defines domain structs (e.g. `model.StockPosition`) with computed fields and summary methods. Handlers in `internal/handler/` manually convert between the two. When adding fields, update both packages.

### Request Flow

All pages use HTMX for dynamic updates without full page reloads:
1. `GET /page` renders the full page template
2. User actions trigger `POST`/`PUT`/`DELETE` that return HTML partials
3. HTMX swaps the partial into the page

### Routes

Routes are defined in `internal/server/server.go`. `/` redirects to `/credit`. Each feature area (`/credit`, `/credit/compare`, `/dashboard`, `/portfolio`, `/budget`, `/tax`) has a handler in `internal/handler/` with a `GET` page route and `POST`/`PUT`/`DELETE` routes for CRUD operations that return HTML partials. The portfolio handler is the most complex, with sub-routes for stocks, crypto, cash, and watchlist.

### External APIs

**Yahoo Finance** (`internal/quote/yahoo.go`):
- ISIN to symbol resolution with exchange preferences by country
- Real-time prices and historical data
- Exchange rate caching (24h TTL)
- Authenticated quoteSummary endpoint (crumb + cookies, cached 1h)

**CoinGecko** (`internal/quote/coingecko.go`):
- Crypto search and price lookup (EUR)
- Batch price fetching with 5-minute cache
- History with singleflight pattern to prevent duplicate requests

**INSEE** (`internal/quote/insee.go`):
- French inflation data from INSEE BDM (SDMX/XML format)
- Annual inflation rates with cumulative multipliers
- 24h cache TTL

### Template System

Templates use `html/template` with custom functions defined in `server.go`:
- `formatMoney` - French currency (space separator)
- `formatDate` - DD/MM/YYYY format
- Arithmetic: `add`, `sub`, `mul`, `div` (float64), `intAdd`, `subInt`, `intDiv`, `mod` (int)
- `seq`, `toJSON`, `percentInRange`

**Critical**: Every page template defines its content with `{{define "content"}}...{{end}}` and **must** end with `{{template "layout" .}}` as the last line outside the define block. Missing this line results in a blank page (1-byte response).

### Data Persistence

SQLite database at `~/.local/share/gostock/gostock.db`:
- `FormInputs` - Credit simulator state (single row, upsert)
- `StockPosition`, `CryptoPosition`, `CashPosition` - Portfolio holdings
- `WatchlistItem` - Watchlist ISINs for monitoring
- `StockSale`, `CryptoSale` - Tax reporting transactions
- `StockPurchase` - Purchase history for PRU (Prix de Revient Unitaire) calculation
- `PortfolioSnapshot` - Daily portfolio value snapshots by asset class
- `BudgetInputs` - Budget/Sankey diagram configuration (single row, upsert)
- `CompareInputs` - Loan offer comparison state (single row, upsert)

The `Store` interface (`persistence/store.go`) defines all persistence operations. Single-page form state (`FormInputs`, `CompareInputs`, `BudgetInputs`) uses a single-row upsert pattern: `INSERT INTO ... ON CONFLICT(id) DO UPDATE SET ...` with `id=1`.

### Key Design Patterns

- **HTMX-driven**: Form submissions return HTML partials, no JSON APIs
- **French localization**: All UI text, number formatting, date formatting
- **Pure calculators**: Calculator package has no side effects, easy to test
- **Monetary precision**: All calculations round to 2 decimal places
- **Rate limiting protection**: Semaphores and caching for external APIs
- **Concurrent data loading**: Handlers use `sync.WaitGroup` goroutines to fetch external data in parallel (e.g. `loadWatchlistSummary`, `loadStockSummary`)

### Additional Documentation

- **`docs/FEATURES.md`** — Detailed French documentation of the credit simulator features (PTZ, PAL, BRS, HCSF rules, rent vs. buy comparison, resale projections)
