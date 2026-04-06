package strategy

import (
	"math"
	"time"

	"stock357/scraper"
)

// AnalysisResult is the full JSON payload returned to the frontend.
type AnalysisResult struct {
	// Stock metadata
	Symbol       string  `json:"symbol"`
	Currency     string  `json:"currency"`
	CurrentPrice float64 `json:"current_price"`
	Week52High   float64 `json:"week_52_high"`
	Week52Low    float64 `json:"week_52_low"`
	ATR14        float64 `json:"atr_14"`
	AnnualizedVol float64 `json:"annualized_vol_pct"`

	// Entry / exit recommendations
	EntryPrice   float64 `json:"entry_price"`
	StopPrice    float64 `json:"stop_price"`
	ProfitTarget float64 `json:"profit_target"`
	StopDistance float64 `json:"stop_distance"`

	// Rule 1 — 2%: position sizing
	MaxRiskDollars  float64 `json:"max_risk_dollars"`
	SuggestedShares int     `json:"suggested_shares"`
	PositionValue   float64 `json:"position_value"`
	ActualRiskPct   float64 `json:"actual_risk_pct"`

	// Rule 2 — 10%: total open risk (5 trades × 2% each)
	MaxOpenRiskDollars float64 `json:"max_open_risk_dollars"`
	MaxSimultTrades    int     `json:"max_simult_trades"`

	// Rule 3 — 7:3 reward:risk
	RewardRiskRatio float64 `json:"reward_risk_ratio"`

	// Financial model
	KellyCriterionPct       float64 `json:"kelly_criterion_pct"`
	ExpectedValuePerShare   float64 `json:"expected_value_per_share"`
	WinRateAssumed          float64 `json:"win_rate_assumed"`
	ExpectedAnnualReturn    float64 `json:"expected_annual_return_pct"`
	TradesPerYear           int     `json:"trades_per_year"`

	// Chart data (last 90 bars)
	ChartDates   []string  `json:"chart_dates"`
	ChartCloses  []float64 `json:"chart_closes"`
	ChartHighs   []float64 `json:"chart_highs"`
	ChartLows    []float64 `json:"chart_lows"`
	ChartVolumes []float64 `json:"chart_volumes"`
	SMA20        []float64 `json:"sma_20"`
	SMA50        []float64 `json:"sma_50"`
}

// Calculate runs all 3-5-7 strategy and financial model calculations.
// entryPrice: use 0 to default to the current market price (sd.CurrentPrice).
// tradesPerYear: expected number of trades per year for the annual return projection.
func Calculate(sd *scraper.StockData, bankroll, entryPrice float64, tradesPerYear int) AnalysisResult {
	atr := calcATR14(sd.Highs, sd.Lows, sd.Closes)
	vol := calcAnnualizedVol(sd.Closes)

	if entryPrice <= 0 {
		entryPrice = sd.CurrentPrice
	}
	entry := entryPrice

	// Stop loss at 2× ATR (Wilder/Kaufman empirical floor — avoids noise-triggered stops)
	stopDist := 2.0 * atr
	stopPrice := entry - stopDist
	// Profit target: 7:3 reward:risk ratio (2.333:1)
	profitTarget := entry + stopDist*(7.0/3.0)

	// Rule 1 — 2% (academic standard; 3% accelerates ruin under losing streaks)
	maxRisk := bankroll * 0.02
	shares := 0
	if stopDist > 0.01 {
		shares = int(math.Floor(maxRisk / stopDist))
	}
	posVal := float64(shares) * entry
	actualRiskPct := 0.0
	if bankroll > 0 && shares > 0 {
		actualRiskPct = (float64(shares) * stopDist) / bankroll * 100
	}

	// Rule 2 — 10% total open risk (5 simultaneous trades × 2% each)
	maxOpenRisk := bankroll * 0.10
	maxSimult := 0
	if maxRisk > 0 {
		maxSimult = int(math.Floor(maxOpenRisk / maxRisk))
	}

	// Financial model (50% win rate — conservative, matches academic literature)
	winRate := 0.50
	rrRatio := 7.0 / 3.0
	// Half Kelly: f* = (W - (1-W)/R) / 2
	// Full Kelly maximizes geometric growth but draws down 30-50%; half-Kelly is the institutional standard.
	fullKelly := winRate - (1-winRate)/rrRatio
	if fullKelly < 0 {
		fullKelly = 0
	}
	kelly := fullKelly / 2.0
	// Expected value per share
	ev := (winRate * (profitTarget - entry)) - ((1 - winRate) * stopDist)
	// Expected annual return based on user-supplied trades/year
	expectedAnnual := (ev * float64(shares) * float64(tradesPerYear)) / bankroll * 100

	// Build SMA over full history, then slice last 90
	sma20full := calcSMA(sd.Closes, 20)
	sma50full := calcSMA(sd.Closes, 50)

	n := len(sd.Closes)
	start := n - 90
	if start < 0 {
		start = 0
	}

	chartDates := make([]string, 0, n-start)
	for _, ts := range sd.Timestamps[start:] {
		chartDates = append(chartDates, time.Unix(ts, 0).UTC().Format("2006-01-02"))
	}

	return AnalysisResult{
		Symbol:        sd.Symbol,
		Currency:      sd.Currency,
		CurrentPrice:  entry,
		Week52High:    sd.Week52High,
		Week52Low:     sd.Week52Low,
		ATR14:         round2(atr),
		AnnualizedVol: round2(vol),

		EntryPrice:   round2(entry),
		StopPrice:    round2(stopPrice),
		ProfitTarget: round2(profitTarget),
		StopDistance: round2(stopDist),

		MaxRiskDollars:  round2(maxRisk),
		SuggestedShares: shares,
		PositionValue:   round2(posVal),
		ActualRiskPct:   round2(actualRiskPct),

		MaxOpenRiskDollars: round2(maxOpenRisk),
		MaxSimultTrades:    maxSimult,
		RewardRiskRatio:    round2(rrRatio),

		KellyCriterionPct:     round2(kelly * 100),
		ExpectedValuePerShare: round2(ev),
		WinRateAssumed:        winRate,
		ExpectedAnnualReturn:  round2(expectedAnnual),
		TradesPerYear:         tradesPerYear,

		ChartDates:   chartDates,
		ChartCloses:  round2Slice(sd.Closes[start:]),
		ChartHighs:   round2Slice(sd.Highs[start:]),
		ChartLows:    round2Slice(sd.Lows[start:]),
		ChartVolumes: sd.Volumes[start:],
		SMA20:        round2Slice(sma20full[start:]),
		SMA50:        round2Slice(sma50full[start:]),
	}
}

// calcATR14 computes Wilder's 14-period Average True Range.
func calcATR14(highs, lows, closes []float64) float64 {
	n := len(closes)
	if n < 2 {
		return 0
	}

	trs := make([]float64, n)
	trs[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hpc := math.Abs(highs[i] - closes[i-1])
		lpc := math.Abs(lows[i] - closes[i-1])
		trs[i] = math.Max(hl, math.Max(hpc, lpc))
	}

	period := 14
	if n < period+1 {
		period = n - 1
	}

	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	for i := period + 1; i < n; i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}
	return atr
}

// calcAnnualizedVol returns annualized volatility as a percentage.
func calcAnnualizedVol(closes []float64) float64 {
	n := len(closes)
	if n < 2 {
		return 0
	}
	returns := make([]float64, n-1)
	for i := 1; i < n; i++ {
		if closes[i-1] > 0 {
			returns[i-1] = math.Log(closes[i] / closes[i-1])
		}
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(returns) - 1)
	return math.Sqrt(variance) * math.Sqrt(252) * 100
}

// calcSMA returns a slice of SMA values (NaN positions filled with 0).
func calcSMA(closes []float64, period int) []float64 {
	n := len(closes)
	sma := make([]float64, n)
	for i := range sma {
		if i < period-1 {
			sma[i] = 0
			continue
		}
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		sma[i] = sum / float64(period)
	}
	return sma
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round2Slice(s []float64) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = round2(v)
	}
	return out
}
