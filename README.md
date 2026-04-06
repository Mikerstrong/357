# 3-5-7 Stock Strategy Analyzer

A risk-managed stock analyzer built around the **3-5-7 trading rules**. Enter a ticker and your bankroll — the app scrapes live Yahoo Finance data, calculates volatility-based entry/exit levels, sizes your position, and scores stocks so you only take high-quality setups.

---

## What It Does

### Single Stock Analysis
Enter any ticker + your bankroll and get:
- **Entry price** — current market price or your own custom entry
- **Stop loss** — placed at `entry − 2 × ATR14` (volatility-scaled, avoids noise-triggered stops)
- **Profit target** — placed at `entry + 2.33 × stop distance` (7:3 reward:risk ratio)
- **Position size** — how many shares to buy risking no more than 2% of bankroll
- **Kelly Criterion** — optimal fraction of bankroll to allocate (half-Kelly, institutional standard)
- **Expected annual return** — projected P&L based on your expected trades per year
- **Price chart** — last 90 trading days with SMA20, SMA50, stop and target reference lines + volume

### Bulk Scan — 70 Stocks at Once
One click scrapes 70 curated tickers across tech, financials, healthcare, energy, retail, defense, and ETFs. Each stock is scored 0–100 across four factors:

| Factor | Weight | What It Measures |
|---|---|---|
| Annual Return | 40 pts | Expected EV × trades/year relative to bankroll |
| Risk Utilization | 25 pts | How efficiently the 2% risk budget is used |
| Vol Fit | 25 pts | Whether ATR-based stop distance is tradeable |
| Position Efficiency | 10 pts | Share count vs. position value ratio |

Top picks are ranked and displayed. Results can be saved to the server or downloaded as JSON.

### Calculator Mode
Already in a trade? Enter the price you paid and get stop/target levels recalculated at your actual entry — not the current market price.

---

## The 3-5-7 Rules

1. **2% per trade** — never risk more than 2% of bankroll on a single position
2. **10% total open risk** — max 5 simultaneous trades × 2% each
3. **7:3 reward:risk** — profit target must be at least 2.33× the stop distance

---

## Stack

```
frontend/   Vue 3 + Vite + Chart.js (port 5173)
backend/    Go 1.22 (port 8080)
```

- Backend scrapes Yahoo Finance v8 JSON API — no API key required
- Stop/target math uses Wilder's ATR14 (14-period smoothed Average True Range)
- FTP upload via `curl` — no external Go dependencies

---

## Running Locally

```bash
# Requires Go + Node installed
./start.sh

# Frontend only (uses mock data)
cd frontend && npm install && npm run dev

# Backend only
cd backend && go run .
```

## FTP Upload

Copy `.env.example` to `.env` and fill in credentials:

```
FTP_HOST=your.ftp.host
FTP_USER=user@domain.com
FTP_PASS=yourpassword
FTP_DIR=/357
```

Results are uploaded as JSON to the configured FTP directory.

---

## API

| Endpoint | Description |
|---|---|
| `GET /api/analyze?ticker=AAPL&bankroll=10000` | Analyze at current market price |
| `GET /api/calc?ticker=AAPL&bankroll=10000&entry=182.50` | Analyze at your entry price |
| `POST /api/save` | Save result JSON to `backend/results/` |
| `POST /api/upload` | FTP the most recent results file |
