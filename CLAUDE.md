# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A 3-5-7 stock strategy analyzer. Enter a ticker + bankroll size → scrapes Yahoo Finance → calculates risk-adjusted entry/exit prices using the 3-5-7 rules → displays financial model and charts. Results can be saved as JSON and uploaded via FTP.

## Running the App

```bash
# Full stack (requires Go installed)
./start.sh

# Frontend only (mocks backend)
cd frontend && npm run dev   # http://localhost:5173

# Backend only
cd backend && go run .       # http://localhost:8080
```

**Go is not installed by default on this machine:**
```bash
sudo snap install go --classic
# or
wget -qO- https://go.dev/dl/go1.22.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
export PATH=$PATH:/usr/local/go/bin
```

## FTP Configuration

Copy `.env.example` to `.env` and fill in FTP credentials:
```bash
FTP_HOST=192.168.0.x
FTP_USER=your_ftp_user
FTP_PASS=your_ftp_password
FTP_DIR=/357     # folder on FTP server
```

FTP upload uses `curl -T` (no external Go deps needed — curl must be on PATH).

## Architecture

```
frontend/ (Vue 3 + Vite, port 5173)
    └── src/App.vue        — all UI, Chart.js charts, fetch calls to /api/*
    └── vite.config.js     — proxies /api → localhost:8080

backend/ (Go, port 8080)
    ├── main.go            — HTTP server, 4 endpoints
    ├── scraper/yahoo.go   — Yahoo Finance v8 JSON API fetcher
    └── strategy/calculator.go  — ATR, 3-5-7 math, Kelly Criterion
```

## API Endpoints

- `GET /api/analyze?ticker=AAPL&bankroll=10000` — fetch + calculate at current market price
- `GET /api/calc?ticker=AAPL&bankroll=10000&entry=182.50` — same but prices the trade at a user-supplied entry (e.g. what they already paid)
- `POST /api/save` — body: AnalysisResult JSON → saves to `backend/results/{TICKER}_{timestamp}.json`
- `POST /api/upload` — FTPs the most recent results file (requires .env)

## The 3-5-7 Rules

1. **2%** — Never risk more than 2% of bankroll on one trade → position size (academic standard; 3% accelerates ruin under losing streaks)
2. **10%** — Total open trade risk capped at 10% → max 5 simultaneous trades × 2% each
3. **7** — Profit target must be 7:3 (2.33:1) reward:risk ratio

Stop loss is placed at `entry − 2 × ATR14`. Profit target at `entry + 2.33 × stop_distance`.

### Why these values
- **2× ATR stop:** Tighter stops (e.g. 1.5×) get triggered by normal price noise ~13% of days in a non-adverse trend. At 2× ATR you're at ~5% — the empirical floor recommended by Kaufman and Kase.
- **2% per trade:** Van Tharp / Elder standard. Allows ~50 consecutive losers before a 64% drawdown. At 3% that drops to ~34 losers.
- **10% total open risk:** 5 trades × 2% each. Be aware that in correlated selloffs (sector or broad market), all 5 positions can draw down simultaneously — size your watchlist diversity accordingly.

## Key Data

- Yahoo Finance URL: `https://query1.finance.yahoo.com/v8/finance/chart/{ticker}?interval=1d&range=1y`
- Requires `User-Agent: Mozilla/5.0` header to avoid 403
- Null OHLCV values come as JSON `null` → use `[]*float64` not `[]float64` in Go structs
- Chart shows last 90 trading days with SMA20, SMA50, stop/target reference lines + volume
