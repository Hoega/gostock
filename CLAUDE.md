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

GoStock is a French personal finance web application built with Go, Chi router, HTMX, and Chart.js. It combines:
- **Mortgage/credit simulation** with French-specific calculations (PTZ, HCSF rules)
- **Portfolio tracking** for stocks, crypto, and cash
- **Tax reporting** for French tax forms (2042-C, 2086)
- **Dashboard** with global wealth overview

### Package Structure

- **cmd/gostock/main.go** - Entry point, flag parsing, graceful shutdown
- **internal/server/** - HTTP server, Chi router, template loading with custom FuncMap
- **internal/handler/** - HTTP handlers: `credit`, `portfolio`, `dashboard`, `tax`
- **internal/calculator/** - Pure calculation logic (amortization, performance, tax)
- **internal/model/** - Data structures and summary computations
- **internal/persistence/** - SQLite storage with Store interface
- **internal/quote/** - Market data: Yahoo Finance (stocks) and CoinGecko (crypto)
- **web/templates/** - Go HTML templates with layout and partials
- **web/static/** - Static assets (JS, CSS)

### Request Flow

All pages use HTMX for dynamic updates without full page reloads:
1. `GET /page` renders the full page template
2. User actions trigger `POST`/`PUT`/`DELETE` that return HTML partials
3. HTMX swaps the partial into the page

### Routes

| Path | Handler | Description |
|------|---------|-------------|
| `/credit` | creditHandler | Mortgage simulator form |
| `/credit/calculate` | creditHandler | Calculate amortization (HTMX partial) |
| `/portfolio` | portfolioHandler | Stock/crypto/cash positions |
| `/portfolio/quote` | portfolioHandler | ISIN lookup via Yahoo Finance |
| `/portfolio/crypto/quote` | portfolioHandler | Crypto lookup via CoinGecko |
| `/portfolio/history/*` | portfolioHandler | Historical price charts |
| `/dashboard` | dashboardHandler | Global wealth overview |
| `/tax` | taxHandler | French tax reporting (2042-C, 2086) |

### External APIs

**Yahoo Finance** (`internal/quote/yahoo.go`):
- ISIN → symbol resolution with exchange preferences by country
- Real-time prices and historical data
- Exchange rate caching (24h TTL)

**CoinGecko** (`internal/quote/coingecko.go`):
- Crypto search and price lookup (EUR)
- Batch price fetching with 5-minute cache
- History with singleflight pattern to prevent duplicate requests

### Template System

Templates use `html/template` with custom functions defined in `server.go`:
- `formatMoney` - French currency (space separator)
- `formatDate` - DD/MM/YYYY format
- `seq`, `toJSON`, `sub`, `add`, `mul`, `div`
- `percentInRange` - Calculate position in a range (for UI meters)

### Data Persistence

SQLite database at `~/.local/share/gostock/gostock.db`:
- `FormInputs` - Credit simulator state (single row, upsert)
- `StockPosition`, `CryptoPosition`, `CashPosition` - Portfolio holdings
- `StockSale`, `CryptoSale` - Tax reporting transactions
- `StockPurchase` - Purchase history for PRU (Prix de Revient Unitaire) calculation

The `Store` interface (`persistence/store.go`) defines all persistence operations.

### Key Design Patterns

- **HTMX-driven**: Form submissions return HTML partials, no JSON APIs
- **French localization**: All UI text, number formatting, date formatting
- **Pure calculators**: Calculator package has no side effects, easy to test
- **Monetary precision**: All calculations round to 2 decimal places
- **Rate limiting protection**: Semaphores and caching for external APIs

### Credit Simulator Features

- Multi-loan support (prêt principal, PTZ, PAL)
- Aid eligibility calculation (PTZ/PAL/BRS by zone and household size)
- HCSF 35% debt ratio enforcement
- Rent vs. buy comparison with opportunity cost
- Resale profitability projections at various appreciation rates

### Portfolio Features

- Stock positions with ISIN-based price lookup
- Crypto positions with CoinGecko integration
- Cash positions with interest rate tracking
- Multi-currency support (USD→EUR conversion)
- Historical performance charts (1m, 3m, 1y, 5y, max)

### Tax Features

- PRU calculation from purchase history
- French stock tax (2042-C) gain/loss computation
- French crypto tax (2086) with portfolio method
