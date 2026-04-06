<template>
  <div class="app">
    <div class="container">
      <header>
        <h1>3-5-7 Strategy Analyzer</h1>
        <p class="subtitle">Risk-adjusted entry &amp; exit recommendations based on bankroll size</p>
      </header>

      <!-- Page Nav -->
      <div class="page-nav">
        <button :class="['nav-tab', currentView === 'analyzer' ? 'active' : '']" @click="currentView = 'analyzer'">Analyzer</button>
        <button :class="['nav-tab', currentView === 'cal' ? 'active' : '']" @click="currentView = 'cal'">Calculator</button>
      </div>

      <!-- ═══ ANALYZER VIEW ═══ -->
      <template v-if="currentView === 'analyzer'">

      <!-- Input Form -->
      <div class="form-card">
        <div class="form-row">
          <div class="field">
            <label>Ticker Symbol</label>
            <input v-model="ticker" type="text" placeholder="AAPL" maxlength="10" @keyup.enter="analyze" />
          </div>
          <div class="field">
            <label>Bankroll ($)</label>
            <input v-model.number="bankroll" type="number" min="100" step="100" placeholder="10000" />
          </div>
          <div class="field">
            <label>Trades / Year</label>
            <input v-model.number="tradesPerYear" type="number" min="1" max="365" step="1" placeholder="20" style="width:100px" />
          </div>
          <button class="btn-primary" :disabled="loading" @click="analyze">
            {{ loading ? 'Fetching…' : 'Analyze' }}
          </button>
          <button class="btn-bulk" :disabled="bulkRunning" @click="bulkScrape">
            {{ bulkRunning ? `Scraping… ${bulkDone}/${SYMBOLS.length}` : 'Scrape 70 Stocks' }}
          </button>
        </div>
      </div>

      <div v-if="error" class="error-banner">{{ error }}</div>

      <!-- Bulk Results -->
      <div v-if="bulkVisible" class="bulk-card">
        <div class="bulk-header">
          <span class="bulk-title">Bulk Scan Results</span>
          <div class="bulk-actions">
            <span class="action-msg">{{ bulkStatus }}</span>
            <button class="btn-secondary" :disabled="!bulkDone || bulkRunning" @click="calcRecommendations">
              Load Recommendations
            </button>
            <button class="btn-secondary" :disabled="!bulkDone || bulkRunning" @click="downloadAll">
              Download JSON
            </button>
            <button class="btn-secondary" :disabled="!bulkDone || bulkRunning || saving" @click="saveToServer">
              {{ saving ? 'Saving…' : 'Save to Server' }}
            </button>
          </div>
        </div>

        <div v-if="bulkRunning" class="progress-wrap">
          <div class="progress-label">{{ bulkDone }} / {{ SYMBOLS.length }}</div>
          <div class="progress-bar-bg">
            <div class="progress-bar-fill" :style="{ width: (bulkDone / SYMBOLS.length * 100) + '%' }"></div>
          </div>
        </div>

        <!-- Recommendations Panel -->
        <div v-if="recommendations.length" class="rec-panel">
          <div class="rec-panel-header">
            <span class="rec-panel-title">Top 3-5-7 Picks</span>
            <span class="rec-panel-sub">Scored across 4 factors · max 100 pts</span>
          </div>
          <div class="score-legend">
            <span class="score-math">
              Score = <b>Annual Return</b> (0–40) + <b>Risk Utilization</b> (0–25) + <b>Vol Fit</b> (0–25) + <b>Position Efficiency</b> (0–10)
            </span>
          </div>
          <div class="rec-list">
            <div
              v-for="(rec, i) in recommendations"
              :key="rec.symbol"
              class="rec-row"
              @click="showDetail(rec.result)"
            >
              <!-- Header row -->
              <div class="rec-row-top">
                <span class="rec-rank">#{{ i + 1 }}</span>
                <span class="rec-sym">{{ rec.symbol }}</span>
                <span class="rec-badge" :class="ratingClass(rec.score)">{{ ratingLabel(rec.score) }}</span>
                <div class="rec-score-bar-wrap">
                  <div class="rec-score-bar" :style="{ width: rec.score + '%' }"></div>
                </div>
                <span class="rec-score-num">{{ rec.score }}<span class="rec-score-denom">/100</span></span>
              </div>
              <!-- Prices row -->
              <div class="rec-row-prices">
                <div class="rec-price-box rec-price-buy">
                  <div class="rec-price-lbl">Buy</div>
                  <div class="rec-price-val">${{ fmt(rec.result.entry_price) }}</div>
                  <div class="rec-price-sub">entry</div>
                </div>
                <div class="rec-price-arrow">→</div>
                <div class="rec-price-box rec-price-stop">
                  <div class="rec-price-lbl">Stop Loss</div>
                  <div class="rec-price-val">${{ fmt(rec.result.stop_price) }}</div>
                  <div class="rec-price-sub td-neg">−{{ fmt(rec.result.stop_distance / rec.result.entry_price * 100) }}%</div>
                </div>
                <div class="rec-price-arrow">→</div>
                <div class="rec-price-box rec-price-target">
                  <div class="rec-price-lbl">Target</div>
                  <div class="rec-price-val">${{ fmt(rec.result.profit_target) }}</div>
                  <div class="rec-price-sub td-pos">+{{ fmt((rec.result.profit_target - rec.result.entry_price) / rec.result.entry_price * 100) }}%</div>
                </div>
                <div class="rec-price-divider"></div>
                <div class="rec-price-box">
                  <div class="rec-price-lbl">Shares</div>
                  <div class="rec-price-val">{{ rec.result.suggested_shares }}</div>
                </div>
                <div class="rec-price-box">
                  <div class="rec-price-lbl">Pos Value</div>
                  <div class="rec-price-val">${{ fmt(rec.result.position_value) }}</div>
                </div>
                <div class="rec-price-box">
                  <div class="rec-price-lbl">EAR</div>
                  <div class="rec-price-val" :class="rec.result.expected_annual_return_pct >= 0 ? 'td-pos' : 'td-neg'">{{ fmt(rec.result.expected_annual_return_pct) }}%</div>
                </div>
              </div>
              <!-- Score breakdown -->
              <div class="rec-breakdown">
                Return&nbsp;<b>{{ rec.components.ret }}/40</b>
                &nbsp;·&nbsp;Risk Util&nbsp;<b>{{ rec.components.risk }}/25</b>
                &nbsp;·&nbsp;Vol Fit&nbsp;<b>{{ rec.components.vol }}/25</b>
                &nbsp;·&nbsp;Position&nbsp;<b>{{ rec.components.pos }}/10</b>
              </div>
            </div>
          </div>
        </div>

        <div class="bulk-table-wrap">
          <table>
            <thead>
              <tr>
                <th>Symbol</th><th>Price</th><th>Entry</th><th>Stop</th><th>Target</th>
                <th>Shares</th><th>Pos Value</th><th>ATR</th><th>Vol %</th><th>EAR %</th><th>Score</th><th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in bulkRows"
                :key="row.symbol"
                :class="{ clickable: row.result }"
                @click="row.result && showDetail(row.result)"
              >
                <td class="td-sym">{{ row.symbol }}</td>
                <template v-if="row.result">
                  <td>${{ fmt(row.result.current_price) }}</td>
                  <td>${{ fmt(row.result.entry_price) }}</td>
                  <td>${{ fmt(row.result.stop_price) }}</td>
                  <td>${{ fmt(row.result.profit_target) }}</td>
                  <td>{{ row.result.suggested_shares }}</td>
                  <td>${{ fmt(row.result.position_value) }}</td>
                  <td>${{ fmt(row.result.atr_14) }}</td>
                  <td>{{ fmt(row.result.annualized_vol_pct) }}%</td>
                  <td :class="row.result.expected_annual_return_pct >= 0 ? 'td-pos' : 'td-neg'">
                    {{ fmt(row.result.expected_annual_return_pct) }}%
                  </td>
                  <td>
                    <span v-if="row.score !== undefined" :class="ratingClass(row.score)" style="font-weight:700">
                      {{ row.score }}
                    </span>
                    <span v-else class="td-muted">—</span>
                  </td>
                  <td class="td-pos">ok</td>
                </template>
                <template v-else-if="row.error">
                  <td v-for="_ in 10" :key="_" class="td-muted">—</td>
                  <td class="td-err">{{ row.error }}</td>
                </template>
                <template v-else>
                  <td v-for="_ in 11" :key="_" class="td-muted">—</td>
                </template>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Single Result -->
      <template v-if="result">
        <div class="rec-card">
          <div class="rec-header">
            <span class="rec-ticker">{{ result.symbol }}</span>
            <span class="rec-price">${{ fmt(result.current_price) }}</span>
            <span class="rec-currency">{{ result.currency }}</span>
          </div>
          <div class="rec-grid">
            <div class="rec-item entry">
              <div class="rec-label">Entry</div>
              <div class="rec-value">${{ fmt(result.entry_price) }}</div>
              <div class="rec-note">Current market price</div>
            </div>
            <div class="rec-item stop">
              <div class="rec-label">Stop Loss</div>
              <div class="rec-value">${{ fmt(result.stop_price) }}</div>
              <div class="rec-note">−${{ fmt(result.stop_distance) }} (2× ATR)</div>
            </div>
            <div class="rec-item target">
              <div class="rec-label">Profit Target</div>
              <div class="rec-value">${{ fmt(result.profit_target) }}</div>
              <div class="rec-note">{{ fmt(result.reward_risk_ratio) }}:1 reward:risk</div>
            </div>
          </div>
        </div>

        <div class="stats-row">
          <div class="stat-card"><div class="stat-label">ATR (14-day)</div><div class="stat-value">${{ fmt(result.atr_14) }}</div></div>
          <div class="stat-card"><div class="stat-label">Annualized Vol</div><div class="stat-value">{{ fmt(result.annualized_vol_pct) }}%</div></div>
          <div class="stat-card"><div class="stat-label">52-Week High</div><div class="stat-value">${{ fmt(result.week_52_high) }}</div></div>
          <div class="stat-card"><div class="stat-label">52-Week Low</div><div class="stat-value">${{ fmt(result.week_52_low) }}</div></div>
        </div>

        <div class="rules-grid">
          <div class="rule-card rule-3">
            <div class="rule-num">2%</div>
            <div class="rule-title">Position Sizing</div>
            <div class="rule-items">
              <div class="rule-row"><span>Max risk</span><span class="val">${{ fmt(result.max_risk_dollars) }}</span></div>
              <div class="rule-row"><span>Suggested shares</span><span class="val">{{ result.suggested_shares }}</span></div>
              <div class="rule-row"><span>Position value</span><span class="val">${{ fmt(result.position_value) }}</span></div>
              <div class="rule-row"><span>Actual risk</span><span class="val">{{ fmt(result.actual_risk_pct) }}%</span></div>
            </div>
          </div>
          <div class="rule-card rule-5">
            <div class="rule-num">10%</div>
            <div class="rule-title">Open Trade Budget</div>
            <div class="rule-items">
              <div class="rule-row"><span>Max total risk</span><span class="val">${{ fmt(result.max_open_risk_dollars) }}</span></div>
              <div class="rule-row"><span>Max simultaneous</span><span class="val">{{ result.max_simult_trades }} trade{{ result.max_simult_trades !== 1 ? 's' : '' }}</span></div>
              <div class="rule-row"><span>Per-trade budget</span><span class="val">${{ fmt(result.max_risk_dollars) }}</span></div>
            </div>
          </div>
          <div class="rule-card rule-7">
            <div class="rule-num">7</div>
            <div class="rule-title">Reward Ratio</div>
            <div class="rule-items">
              <div class="rule-row"><span>Reward : Risk</span><span class="val">{{ fmt(result.reward_risk_ratio) }} : 1</span></div>
              <div class="rule-row"><span>Target gain</span><span class="val">+${{ fmt(result.profit_target - result.entry_price) }}</span></div>
              <div class="rule-row"><span>Max loss</span><span class="val">−${{ fmt(result.stop_distance) }}</span></div>
            </div>
          </div>
        </div>

        <div class="model-card">
          <h2 class="section-title">Financial Model</h2>
          <div class="model-grid">
            <div class="model-item">
              <div class="model-label">Kelly Criterion</div>
              <div class="model-value">{{ fmt(result.kelly_criterion_pct) }}%</div>
              <div class="model-note">Half Kelly — optimal fraction of bankroll</div>
            </div>
            <div class="model-item">
              <div class="model-label">Expected Value / Share</div>
              <div class="model-value" :class="result.expected_value_per_share >= 0 ? 'pos' : 'neg'">
                ${{ fmt(result.expected_value_per_share) }}
              </div>
              <div class="model-note">At 50% assumed win rate</div>
            </div>
            <div class="model-item">
              <div class="model-label">Expected Annual Return</div>
              <div class="model-value" :class="result.expected_annual_return_pct >= 0 ? 'pos' : 'neg'">
                {{ fmt(result.expected_annual_return_pct) }}%
              </div>
              <div class="model-note">~{{ result.trades_per_year }} trades/year at this setup</div>
            </div>
            <div class="model-item">
              <div class="model-label">Position as % of Bankroll</div>
              <div class="model-value">{{ fmt(result.position_value / bankroll * 100) }}%</div>
              <div class="model-note">${{ fmt(result.position_value) }} of ${{ fmt(bankroll) }}</div>
            </div>
          </div>
        </div>

        <div class="chart-card">
          <h2 class="section-title">{{ result.symbol }} — 90-Day Price History</h2>
          <div class="chart-wrap"><canvas ref="priceCanvas"></canvas></div>
          <div class="chart-wrap volume-wrap"><canvas ref="volCanvas"></canvas></div>
        </div>

        <div class="actions-row">
          <button class="btn-secondary" @click="saveResult">Save as JSON</button>
          <span v-if="actionMsg" class="action-msg">{{ actionMsg }}</span>
        </div>
      </template>

      </template><!-- end analyzer view -->

      <!-- ═══ CALCULATOR VIEW ═══ -->
      <template v-if="currentView === 'cal'">
        <div class="form-card">
          <p class="cal-desc">Enter the ticker and the price you paid. The calculator fetches live ATR data and sets your stop loss and profit target using the 3-5-7 rules.</p>
          <div class="form-row" style="margin-top:1rem">
            <div class="field">
              <label>Ticker Symbol</label>
              <input v-model="calTicker" type="text" placeholder="AAPL" maxlength="10" @keyup.enter="calcAnalyze" />
            </div>
            <div class="field">
              <label>Price I Paid ($)</label>
              <input v-model.number="calPrice" type="number" min="0.01" step="0.01" placeholder="150.00" style="width:140px" />
            </div>
            <div class="field">
              <label>Bankroll ($)</label>
              <input v-model.number="bankroll" type="number" min="100" step="100" placeholder="10000" />
            </div>
            <div class="field">
              <label>Trades / Year</label>
              <input v-model.number="tradesPerYear" type="number" min="1" max="365" step="1" placeholder="20" style="width:100px" />
            </div>
            <button class="btn-primary" :disabled="calLoading" @click="calcAnalyze">
              {{ calLoading ? 'Fetching…' : 'Calculate' }}
            </button>
          </div>
        </div>

        <div v-if="calError" class="error-banner">{{ calError }}</div>

        <template v-if="calResult">
          <!-- Entry vs Current banner -->
          <div class="cal-banner">
            <div class="cal-banner-item">
              <span class="cal-banner-lbl">Your Entry</span>
              <span class="cal-banner-val accent">${{ fmt(calResult.entry_price) }}</span>
            </div>
            <div class="cal-banner-divider"></div>
            <div class="cal-banner-item">
              <span class="cal-banner-lbl">Current Price</span>
              <span class="cal-banner-val">${{ fmt(calResult.current_price) }}</span>
            </div>
            <div class="cal-banner-divider"></div>
            <div class="cal-banner-item">
              <span class="cal-banner-lbl">P&amp;L Since Entry</span>
              <span class="cal-banner-val" :class="calResult.current_price >= calResult.entry_price ? 'td-pos' : 'td-neg'">
                {{ calResult.current_price >= calResult.entry_price ? '+' : '' }}{{ fmt((calResult.current_price - calResult.entry_price) / calResult.entry_price * 100) }}%
              </span>
            </div>
            <div class="cal-banner-divider"></div>
            <div class="cal-banner-item">
              <span class="cal-banner-lbl">ATR (14-day)</span>
              <span class="cal-banner-val">${{ fmt(calResult.atr_14) }}</span>
            </div>
          </div>

          <!-- Stop / Target cards -->
          <div class="rec-card">
            <div class="rec-header">
              <span class="rec-ticker">{{ calResult.symbol }}</span>
              <span class="rec-currency">{{ calResult.currency }}</span>
            </div>
            <div class="rec-grid">
              <div class="rec-item entry">
                <div class="rec-label">Your Entry</div>
                <div class="rec-value">${{ fmt(calResult.entry_price) }}</div>
                <div class="rec-note">Price you paid</div>
              </div>
              <div class="rec-item stop">
                <div class="rec-label">Stop Loss</div>
                <div class="rec-value">${{ fmt(calResult.stop_price) }}</div>
                <div class="rec-note">−${{ fmt(calResult.stop_distance) }} · 2× ATR · −{{ fmt(calResult.stop_distance / calResult.entry_price * 100) }}%</div>
              </div>
              <div class="rec-item target">
                <div class="rec-label">Profit Target</div>
                <div class="rec-value">${{ fmt(calResult.profit_target) }}</div>
                <div class="rec-note">+${{ fmt(calResult.profit_target - calResult.entry_price) }} · {{ fmt(calResult.reward_risk_ratio) }}:1 R:R · +{{ fmt((calResult.profit_target - calResult.entry_price) / calResult.entry_price * 100) }}%</div>
              </div>
            </div>
          </div>

          <!-- Position sizing -->
          <div class="rules-grid" style="grid-template-columns: repeat(2,1fr)">
            <div class="rule-card rule-3">
              <div class="rule-num">2%</div>
              <div class="rule-title">Position Sizing</div>
              <div class="rule-items">
                <div class="rule-row"><span>Max risk</span><span class="val">${{ fmt(calResult.max_risk_dollars) }}</span></div>
                <div class="rule-row"><span>Suggested shares</span><span class="val">{{ calResult.suggested_shares }}</span></div>
                <div class="rule-row"><span>Position value</span><span class="val">${{ fmt(calResult.position_value) }}</span></div>
                <div class="rule-row"><span>Actual risk</span><span class="val">{{ fmt(calResult.actual_risk_pct) }}%</span></div>
              </div>
            </div>
            <div class="rule-card rule-7">
              <div class="rule-num">7:3</div>
              <div class="rule-title">Reward : Risk</div>
              <div class="rule-items">
                <div class="rule-row"><span>If target hit</span><span class="val td-pos">+${{ fmt((calResult.profit_target - calResult.entry_price) * calResult.suggested_shares) }}</span></div>
                <div class="rule-row"><span>If stop hit</span><span class="val td-neg">−${{ fmt(calResult.stop_distance * calResult.suggested_shares) }}</span></div>
                <div class="rule-row"><span>Expected value/share</span><span class="val" :class="calResult.expected_value_per_share >= 0 ? 'td-pos' : 'td-neg'">${{ fmt(calResult.expected_value_per_share) }}</span></div>
                <div class="rule-row"><span>Half Kelly %</span><span class="val">{{ fmt(calResult.kelly_criterion_pct) }}%</span></div>
              </div>
            </div>
          </div>

          <!-- Chart -->
          <div class="chart-card">
            <h2 class="section-title">{{ calResult.symbol }} — 90-Day Price History</h2>
            <div class="chart-wrap"><canvas ref="calPriceCanvas"></canvas></div>
            <div class="chart-wrap volume-wrap"><canvas ref="calVolCanvas"></canvas></div>
          </div>
        </template>
      </template><!-- end cal view -->

    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import {
  Chart, LineController, LineElement, PointElement, LinearScale, CategoryScale,
  BarController, BarElement, Filler, Legend, Tooltip,
} from 'chart.js'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale,
               BarController, BarElement, Filler, Legend, Tooltip)

// ── Stock list ─────────────────────────────────────────────────────────────
const BULK_SYMBOLS = [
  'AAPL','MSFT','NVDA','GOOGL','AMZN','META','TSLA','AMD','INTC','ORCL',
  'ADBE','CRM','NFLX','QCOM','TXN',
  'JPM','BAC','GS','MS','WFC','V','MA','AXP','C','BLK',
  'JNJ','PFE','UNH','ABBV','MRK','LLY','TMO','ABT','BMY','AMGN',
  'XOM','CVX','COP','SLB','EOG',
  'WMT','TGT','COST','HD','LOW','MCD','SBUX','NKE','DIS','PYPL',
  'BA','CAT','GE','HON','UPS','FDX','RTX','LMT','DE',
  'T','VZ','CMCSA',
  'SPY','QQQ','DIA','IWM','GLD',
  'SHOP','SNOW','PLTR','COIN','ROKU','UBER','LYFT','SOFI','RIVN','LCID',
]
const SYMBOLS = [...new Set(BULK_SYMBOLS)].slice(0, 70)

// ── State ──────────────────────────────────────────────────────────────────
const currentView  = ref('analyzer')
const ticker       = ref('')
const bankroll     = ref(10000)
const tradesPerYear= ref(20)
const loading      = ref(false)
const error        = ref(null)
const result       = ref(null)
const actionMsg    = ref('')
const priceCanvas  = ref(null)
const volCanvas    = ref(null)
let priceChart = null, volChart = null

// Cal page state
const calTicker      = ref('')
const calPrice       = ref(0)
const calLoading     = ref(false)
const calError       = ref(null)
const calResult      = ref(null)
const calPriceCanvas = ref(null)
const calVolCanvas   = ref(null)
let calPriceChart = null, calVolChart = null

const bulkRunning    = ref(false)
const bulkVisible    = ref(false)
const bulkDone       = ref(0)
const bulkStatus     = ref('')
const bulkRows       = ref([])
const recommendations = ref([])
const saving         = ref(false)
let   bulkResults    = []

// ── Calculations ───────────────────────────────────────────────────────────
function calcATR14(highs, lows, closes) {
  const n = closes.length
  if (n < 2) return 0
  const trs = new Array(n)
  trs[0] = highs[0] - lows[0]
  for (let i = 1; i < n; i++)
    trs[i] = Math.max(highs[i]-lows[i], Math.abs(highs[i]-closes[i-1]), Math.abs(lows[i]-closes[i-1]))
  let period = Math.min(14, n-1), sum = 0
  for (let i = 1; i <= period; i++) sum += trs[i]
  let atr = sum / period
  for (let i = period+1; i < n; i++) atr = (atr*(period-1) + trs[i]) / period
  return atr
}

function calcAnnualizedVol(closes) {
  const n = closes.length
  if (n < 2) return 0
  const rets = []
  for (let i = 1; i < n; i++) if (closes[i-1] > 0) rets.push(Math.log(closes[i]/closes[i-1]))
  const mean = rets.reduce((a,b) => a+b, 0) / rets.length
  const variance = rets.reduce((s,r) => s+(r-mean)**2, 0) / (rets.length-1)
  return Math.sqrt(variance) * Math.sqrt(252) * 100
}

function calcSMA(closes, period) {
  const n = closes.length, sma = new Array(n).fill(0)
  for (let i = period-1; i < n; i++) {
    let s = 0
    for (let j = i-period+1; j <= i; j++) s += closes[j]
    sma[i] = s / period
  }
  return sma
}

function r2(v) { return Math.round(v*100)/100 }
function r2s(a) { return a.map(r2) }

// entryOverride: pass a price > 0 to use instead of sd.currentPrice (cal page)
// tpy: trades per year for annual return projection
function calculate(sd, br, tpy = 20, entryOverride = 0) {
  const atr = calcATR14(sd.highs, sd.lows, sd.closes)
  const vol = calcAnnualizedVol(sd.closes)
  const entry = entryOverride > 0 ? entryOverride : sd.currentPrice
  // 2× ATR stop — empirical floor to avoid noise-triggered stops (Kaufman, Wilder)
  const stopDist = 2.0 * atr
  const stopPrice = entry - stopDist
  const profitTarget = entry + stopDist * (7/3)
  // 2% per trade — academic standard (Van Tharp); 3% accelerates ruin under streaks
  const maxRisk = br * 0.02
  const sharesFromRisk = stopDist > 0.01 ? Math.floor(maxRisk / stopDist) : 0
  const sharesAffordable = entry > 0 ? Math.floor(br / entry) : 0
  const shares = Math.min(sharesFromRisk, sharesAffordable)
  const posVal = shares * entry
  const actualRiskPct = (br > 0 && shares > 0) ? (shares * stopDist) / br * 100 : 0
  // 10% total = 5 simultaneous trades × 2% each
  const maxOpenRisk = br * 0.10
  const maxSimult = maxRisk > 0 ? Math.floor(maxOpenRisk / maxRisk) : 0
  // 50% win rate — conservative academic baseline (Barber & Odean, 2000)
  const winRate = 0.50, rrRatio = 7/3
  // Half Kelly (Ed Thorp standard — full Kelly draws down 30-50%)
  const fullKelly = Math.max(0, winRate - (1 - winRate) / rrRatio)
  const kelly = fullKelly / 2
  const ev = (winRate * (profitTarget - entry)) - ((1 - winRate) * stopDist)
  const expectedAnnual = (ev * shares * tpy) / br * 100
  const n = sd.closes.length, start = Math.max(0, n - 90)
  const sma20 = calcSMA(sd.closes, 20), sma50 = calcSMA(sd.closes, 50)
  const chartDates = sd.timestamps.slice(start).map(ts => new Date(ts*1000).toISOString().slice(0,10))
  return {
    symbol: sd.symbol, currency: sd.currency, current_price: sd.currentPrice,
    week_52_high: sd.week52High, week_52_low: sd.week52Low,
    atr_14: r2(atr), annualized_vol_pct: r2(vol),
    entry_price: r2(entry), stop_price: r2(stopPrice), profit_target: r2(profitTarget), stop_distance: r2(stopDist),
    max_risk_dollars: r2(maxRisk), suggested_shares: shares, position_value: r2(posVal), actual_risk_pct: r2(actualRiskPct),
    max_open_risk_dollars: r2(maxOpenRisk), max_simult_trades: maxSimult, reward_risk_ratio: r2(rrRatio),
    kelly_criterion_pct: r2(kelly*100), expected_value_per_share: r2(ev), win_rate_assumed: winRate,
    expected_annual_return_pct: r2(expectedAnnual), trades_per_year: tpy,
    chart_dates: chartDates,
    chart_closes: r2s(sd.closes.slice(start)), chart_highs: r2s(sd.highs.slice(start)),
    chart_lows: r2s(sd.lows.slice(start)), chart_volumes: sd.volumes.slice(start),
    sma_20: r2s(sma20.slice(start)), sma_50: r2s(sma50.slice(start)),
  }
}

// ── Yahoo parsing ──────────────────────────────────────────────────────────
function parseYahoo(data) {
  if (data.chart?.error) throw new Error(data.chart.error.description || 'Yahoo error')
  if (!data.chart?.result?.length) throw new Error('No data — check ticker')
  const r = data.chart.result[0], q = r.indicators?.quote?.[0], meta = r.meta
  if (!q) throw new Error('No quote data')
  const timestamps=[], opens=[], highs=[], lows=[], closes=[], volumes=[]
  for (let i = 0; i < r.timestamp.length; i++) {
    const h=q.high?.[i], l=q.low?.[i], c=q.close?.[i]
    if (h==null||l==null||c==null||h===0||l===0||c===0) continue
    timestamps.push(r.timestamp[i]); opens.push(q.open?.[i]??0)
    highs.push(h); lows.push(l); closes.push(c); volumes.push(q.volume?.[i]??0)
  }
  if (closes.length < 15) throw new Error('Insufficient history')
  return { symbol: meta.symbol, currency: meta.currency||'USD', currentPrice: meta.regularMarketPrice,
           week52High: meta.fiftyTwoWeekHigh, week52Low: meta.fiftyTwoWeekLow,
           timestamps, opens, highs, lows, closes, volumes }
}

async function fetchAndCalc(sym, br, tpy = 20, entryOverride = 0) {
  const res = await fetch(`proxy.php?ticker=${encodeURIComponent(sym)}`)
  const data = await res.json()
  if (data.error) throw new Error(data.error)
  return calculate(parseYahoo(data), br, tpy, entryOverride)
}

// ── Single analyze ─────────────────────────────────────────────────────────
async function analyze() {
  const t = ticker.value.trim().toUpperCase()
  if (!t) { error.value = 'Enter a ticker symbol'; return }
  if (!bankroll.value || bankroll.value <= 0) { error.value = 'Enter a valid bankroll amount'; return }
  loading.value = true; error.value = null; result.value = null
  try {
    const r = await fetchAndCalc(t, bankroll.value, tradesPerYear.value)
    result.value = r
    await nextTick()
    renderCharts(r)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

// ── Cal page analyze ────────────────────────────────────────────────────────
async function calcAnalyze() {
  const t = calTicker.value.trim().toUpperCase()
  if (!t) { calError.value = 'Enter a ticker symbol'; return }
  if (!calPrice.value || calPrice.value <= 0) { calError.value = 'Enter the price you paid'; return }
  if (!bankroll.value || bankroll.value <= 0) { calError.value = 'Enter a valid bankroll amount'; return }
  calLoading.value = true; calError.value = null; calResult.value = null
  try {
    const r = await fetchAndCalc(t, bankroll.value, tradesPerYear.value, calPrice.value)
    calResult.value = r
    await nextTick()
    renderCalCharts(r)
  } catch (e) {
    calError.value = e.message
  } finally {
    calLoading.value = false
  }
}

// ── Bulk scrape ────────────────────────────────────────────────────────────
async function bulkScrape() {
  if (!bankroll.value || bankroll.value <= 0) { error.value = 'Enter a valid bankroll amount first'; return }
  error.value = null
  bulkResults = []
  bulkDone.value = 0
  bulkStatus.value = ''
  bulkRunning.value = true
  bulkVisible.value = true
  bulkRows.value = SYMBOLS.map(s => ({ symbol: s, result: null, error: null }))

  const queue = [...SYMBOLS]
  async function worker() {
    while (queue.length) {
      const sym = queue.shift()
      const rowIdx = SYMBOLS.indexOf(sym)
      try {
        const r = await fetchAndCalc(sym, bankroll.value)
        bulkResults.push(r)
        bulkRows.value[rowIdx] = { symbol: sym, result: r, error: null }
      } catch (e) {
        bulkResults.push({ symbol: sym, error: e.message })
        bulkRows.value[rowIdx] = { symbol: sym, result: null, error: e.message }
      }
      bulkDone.value++
    }
  }
  await Promise.all(Array.from({ length: 5 }, () => worker()))
  const ok = bulkResults.filter(r => !r.error).length
  bulkStatus.value = `Done — ${ok}/${SYMBOLS.length} succeeded`
  bulkRunning.value = false
}

function showDetail(r) {
  result.value = r
  nextTick(() => {
    renderCharts(r)
    document.getElementById('detail')?.scrollIntoView({ behavior: 'smooth' })
  })
}

function downloadAll() {
  const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  const blob = new Blob([JSON.stringify(bulkResults, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href = url; a.download = `357_bulk_${ts}.json`; a.click()
  URL.revokeObjectURL(url)
}

async function saveToServer() {
  saving.value = true
  bulkStatus.value = 'Saving to server…'
  try {
    const res = await fetch('upload.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(bulkResults),
    })
    const data = await res.json()
    if (data.error) throw new Error(data.error)
    bulkStatus.value = `Saved: ${data.file}`
  } catch (e) {
    bulkStatus.value = `Save failed: ${e.message}`
  } finally {
    saving.value = false
  }
}

// ── Scoring ────────────────────────────────────────────────────────────────
// Score = Annual Return (0-40) + Risk Utilization (0-25) + Vol Fit (0-25) + Position Efficiency (0-10)
function score357(r) {
  // 1. Expected Annual Return — target 30%+, capped at 40 pts
  const retScore = Math.round(Math.min(40, Math.max(0, r.expected_annual_return_pct / 30 * 40)))

  // 2. Risk Utilization — how close actual_risk_pct is to the full 2% budget
  // Perfect = 2%, diminishes if far below (undersized position)
  const riskScore = Math.round(Math.min(25, Math.max(0, r.actual_risk_pct / 2 * 25)))

  // 3. Volatility Fit — sweet spot 20–45% annualized vol for trend-following
  // <15%: too quiet (small ATR → few shares, low EV)
  // 20–45%: ideal
  // >65%: too erratic (stop gets hit frequently)
  const vol = r.annualized_vol_pct
  let volScore
  if (vol < 15)      volScore = Math.round(vol / 15 * 15)
  else if (vol <= 45) volScore = 25
  else if (vol <= 65) volScore = Math.round(25 - ((vol - 45) / 20) * 10)
  else               volScore = Math.round(Math.max(0, 15 - ((vol - 65) / 35) * 15))

  // 4. Position Efficiency — more shares = finer position control; target 50+ shares = full 10 pts
  const posScore = Math.round(Math.min(10, r.suggested_shares / 50 * 10))

  const total = retScore + riskScore + volScore + posScore
  return {
    total,
    components: { ret: retScore, risk: riskScore, vol: volScore, pos: posScore }
  }
}

function ratingLabel(score) {
  if (score >= 75) return '★★★ STRONG'
  if (score >= 50) return '★★ GOOD'
  if (score >= 30) return '★ MODERATE'
  return 'WEAK'
}

function ratingClass(score) {
  if (score >= 75) return 'rating-strong'
  if (score >= 50) return 'rating-good'
  if (score >= 30) return 'rating-mod'
  return 'rating-weak'
}

function calcRecommendations() {
  const scored = bulkRows.value
    .filter(row => row.result)
    .map(row => {
      const { total, components } = score357(row.result)
      return { symbol: row.symbol, score: total, components, result: row.result }
    })
    .sort((a, b) => b.score - a.score)
    .slice(0, 10)

  recommendations.value = scored

  // Also stamp scores back onto table rows
  bulkRows.value = bulkRows.value.map(row => {
    if (!row.result) return row
    const { total } = score357(row.result)
    return { ...row, score: total }
  })
}

// ── Charts ─────────────────────────────────────────────────────────────────
function renderCharts(data) {
  if (priceChart) priceChart.destroy()
  if (volChart)   volChart.destroy()
  const gc = '#2a2a2a', tc = '#8b949e'
  const stopLine   = Array(data.chart_dates.length).fill(data.stop_price)
  const targetLine = Array(data.chart_dates.length).fill(data.profit_target)
  const sma20 = data.sma_20.map(v => v===0 ? null : v)
  const sma50 = data.sma_50.map(v => v===0 ? null : v)
  priceChart = new Chart(priceCanvas.value, {
    type: 'line',
    data: { labels: data.chart_dates, datasets: [
      { label: `${data.symbol} Close`, data: data.chart_closes, borderColor:'#4fc3f7', backgroundColor:'rgba(79,195,247,0.05)', borderWidth:2, pointRadius:0, fill:true, tension:0.1, order:1 },
      { label:'SMA 20', data:sma20, borderColor:'#ffa726', borderWidth:1.5, pointRadius:0, fill:false, tension:0.1, spanGaps:true, order:2 },
      { label:'SMA 50', data:sma50, borderColor:'#ab47bc', borderWidth:1.5, pointRadius:0, fill:false, tension:0.1, spanGaps:true, order:3 },
      { label:'Stop Loss', data:stopLine, borderColor:'#ef5350', borderWidth:1, borderDash:[6,4], pointRadius:0, fill:false, order:4 },
      { label:'Target', data:targetLine, borderColor:'#66bb6a', borderWidth:1, borderDash:[6,4], pointRadius:0, fill:false, order:5 },
    ]},
    options: { responsive:true, maintainAspectRatio:false, interaction:{mode:'index',intersect:false},
      plugins:{ legend:{labels:{color:'#e0e0e0',boxWidth:12,font:{size:11}}}, tooltip:{backgroundColor:'#1c2128',titleColor:'#e6edf3',bodyColor:'#8b949e'} },
      scales:{ x:{ticks:{color:tc,maxTicksLimit:10},grid:{color:gc}}, y:{ticks:{color:tc},grid:{color:gc}} },
    },
  })
  volChart = new Chart(volCanvas.value, {
    type: 'bar',
    data: { labels: data.chart_dates, datasets: [{ label:'Volume', data:data.chart_volumes, backgroundColor:'rgba(79,195,247,0.25)', borderWidth:0 }] },
    options: { responsive:true, maintainAspectRatio:false,
      plugins:{ legend:{labels:{color:'#e0e0e0',boxWidth:12,font:{size:11}}}, tooltip:{backgroundColor:'#1c2128',titleColor:'#e6edf3',bodyColor:'#8b949e'} },
      scales:{ x:{ticks:{color:tc,maxTicksLimit:10},grid:{color:gc}}, y:{ticks:{color:tc},grid:{color:gc}} },
    },
  })
}

function renderCalCharts(data) {
  if (calPriceChart) calPriceChart.destroy()
  if (calVolChart)   calVolChart.destroy()
  const gc = '#2a2a2a', tc = '#8b949e'
  const stopLine   = Array(data.chart_dates.length).fill(data.stop_price)
  const targetLine = Array(data.chart_dates.length).fill(data.profit_target)
  const entryLine  = Array(data.chart_dates.length).fill(data.entry_price)
  const sma20 = data.sma_20.map(v => v===0 ? null : v)
  const sma50 = data.sma_50.map(v => v===0 ? null : v)
  calPriceChart = new Chart(calPriceCanvas.value, {
    type: 'line',
    data: { labels: data.chart_dates, datasets: [
      { label: `${data.symbol} Close`, data: data.chart_closes, borderColor:'#4fc3f7', backgroundColor:'rgba(79,195,247,0.05)', borderWidth:2, pointRadius:0, fill:true, tension:0.1, order:1 },
      { label:'SMA 20', data:sma20, borderColor:'#ffa726', borderWidth:1.5, pointRadius:0, fill:false, tension:0.1, spanGaps:true, order:2 },
      { label:'SMA 50', data:sma50, borderColor:'#ab47bc', borderWidth:1.5, pointRadius:0, fill:false, tension:0.1, spanGaps:true, order:3 },
      { label:'Your Entry', data:entryLine, borderColor:'#4fc3f7', borderWidth:1.5, borderDash:[4,3], pointRadius:0, fill:false, order:4 },
      { label:'Stop Loss', data:stopLine, borderColor:'#ef5350', borderWidth:1, borderDash:[6,4], pointRadius:0, fill:false, order:5 },
      { label:'Target', data:targetLine, borderColor:'#66bb6a', borderWidth:1, borderDash:[6,4], pointRadius:0, fill:false, order:6 },
    ]},
    options: { responsive:true, maintainAspectRatio:false, interaction:{mode:'index',intersect:false},
      plugins:{ legend:{labels:{color:'#e0e0e0',boxWidth:12,font:{size:11}}}, tooltip:{backgroundColor:'#1c2128',titleColor:'#e6edf3',bodyColor:'#8b949e'} },
      scales:{ x:{ticks:{color:tc,maxTicksLimit:10},grid:{color:gc}}, y:{ticks:{color:tc},grid:{color:gc}} },
    },
  })
  calVolChart = new Chart(calVolCanvas.value, {
    type: 'bar',
    data: { labels: data.chart_dates, datasets: [{ label:'Volume', data:data.chart_volumes, backgroundColor:'rgba(79,195,247,0.25)', borderWidth:0 }] },
    options: { responsive:true, maintainAspectRatio:false,
      plugins:{ legend:{labels:{color:'#e0e0e0',boxWidth:12,font:{size:11}}}, tooltip:{backgroundColor:'#1c2128',titleColor:'#e6edf3',bodyColor:'#8b949e'} },
      scales:{ x:{ticks:{color:tc,maxTicksLimit:10},grid:{color:gc}}, y:{ticks:{color:tc},grid:{color:gc}} },
    },
  })
}

function saveResult() {
  if (!result.value) return
  const ts = new Date().toISOString().replace(/[:.]/g,'-').slice(0,19)
  const blob = new Blob([JSON.stringify(result.value, null, 2)], { type:'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href=url; a.download=`${result.value.symbol}_${ts}.json`; a.click()
  URL.revokeObjectURL(url)
  actionMsg.value = `Downloaded: ${result.value.symbol}_${ts}.json`
}

const fmt = n => n==null ? '—' : Math.abs(n)>=1000
  ? n.toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2})
  : n.toFixed(2)
</script>

<style>
:root {
  --bg:#0d1117; --surface:#161b22; --surface2:#21262d; --border:#30363d;
  --text:#e6edf3; --muted:#8b949e; --accent:#4fc3f7;
  --danger:#ef5350; --success:#66bb6a; --warn:#ffa726; --purple:#ab47bc;
}
*{box-sizing:border-box;margin:0;padding:0;}
body{background:var(--bg);color:var(--text);font-family:'Segoe UI',system-ui,sans-serif;font-size:14px;line-height:1.5;}
.app{min-height:100vh;padding:2rem 1rem;}
.container{max-width:1100px;margin:0 auto;}
header{margin-bottom:1.5rem;}
h1{font-size:1.75rem;color:var(--accent);font-weight:700;}
.subtitle{color:var(--muted);margin-top:.25rem;}

/* Page nav */
.page-nav{display:flex;gap:.5rem;margin-bottom:1.5rem;border-bottom:1px solid var(--border);padding-bottom:.75rem;}
.nav-tab{background:none;border:1px solid transparent;color:var(--muted);padding:.45rem 1.1rem;border-radius:6px;font-size:.9rem;font-weight:600;cursor:pointer;transition:all .15s;}
.nav-tab:hover{color:var(--text);}
.nav-tab.active{background:var(--surface);border-color:var(--border);color:var(--accent);}

.form-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.25rem 1.5rem;margin-bottom:1.5rem;}
.form-row{display:flex;gap:1rem;flex-wrap:wrap;align-items:flex-end;}
.field{display:flex;flex-direction:column;gap:.35rem;}
label{font-size:.72rem;color:var(--muted);text-transform:uppercase;letter-spacing:.06em;font-weight:600;}
input{background:var(--surface2);border:1px solid var(--border);color:var(--text);padding:.55rem .85rem;border-radius:6px;font-size:.95rem;outline:none;transition:border-color .15s;}
input:focus{border-color:var(--accent);}
input[type=text]{text-transform:uppercase;width:120px;}
input[type=number]{width:160px;}

.btn-primary{background:var(--accent);color:#000;border:none;padding:.6rem 1.5rem;border-radius:6px;font-size:.95rem;font-weight:700;cursor:pointer;transition:opacity .15s;align-self:flex-end;}
.btn-primary:hover:not(:disabled){opacity:.85;}
.btn-primary:disabled{opacity:.4;cursor:not-allowed;}

.btn-bulk{background:var(--warn);color:#000;border:none;padding:.6rem 1.5rem;border-radius:6px;font-size:.95rem;font-weight:700;cursor:pointer;transition:opacity .15s;align-self:flex-end;}
.btn-bulk:hover:not(:disabled){opacity:.85;}
.btn-bulk:disabled{opacity:.4;cursor:not-allowed;}

.error-banner{background:rgba(239,83,80,.12);border:1px solid var(--danger);color:var(--danger);padding:.75rem 1rem;border-radius:8px;margin-bottom:1.5rem;}

/* Bulk */
.bulk-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.5rem;margin-bottom:1.5rem;}
.bulk-header{display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem;flex-wrap:wrap;gap:.75rem;}
.bulk-title{font-size:1rem;font-weight:700;}
.bulk-actions{display:flex;gap:.75rem;align-items:center;flex-wrap:wrap;}
.progress-wrap{margin-bottom:1rem;}
.progress-label{font-size:.8rem;color:var(--muted);margin-bottom:.4rem;}
.progress-bar-bg{background:var(--surface2);border-radius:999px;height:8px;overflow:hidden;}
.progress-bar-fill{height:100%;background:var(--warn);border-radius:999px;transition:width .2s;}
.bulk-table-wrap{overflow-x:auto;}
table{width:100%;border-collapse:collapse;font-size:.82rem;}
thead th{text-align:left;padding:.5rem .75rem;color:var(--muted);font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;border-bottom:1px solid var(--border);white-space:nowrap;}
tbody tr{border-bottom:1px solid var(--border);transition:background .1s;}
tbody tr.clickable{cursor:pointer;}
tbody tr.clickable:hover{background:var(--surface2);}
tbody td{padding:.45rem .75rem;white-space:nowrap;}
.td-sym{font-weight:700;color:var(--accent);}
.td-pos{color:var(--success);}
.td-neg{color:var(--danger);}
.td-err{color:var(--muted);font-style:italic;}
.td-muted{color:var(--muted);}

/* Recommendations */
.rec-panel{background:var(--surface2);border:1px solid var(--border);border-radius:10px;padding:1.25rem;margin-bottom:1.25rem;}
.rec-panel-header{display:flex;align-items:baseline;gap:1rem;margin-bottom:.5rem;flex-wrap:wrap;}
.rec-panel-title{font-size:1rem;font-weight:700;}
.rec-panel-sub{font-size:.75rem;color:var(--muted);}
.score-legend{margin-bottom:1rem;}
.score-math{font-size:.75rem;color:var(--muted);}
.score-math b{color:var(--text);}
.rec-list{display:flex;flex-direction:column;gap:.5rem;}
.rec-row{display:flex;flex-direction:column;gap:.6rem;background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:1rem 1.25rem;cursor:pointer;transition:border-color .15s;}
.rec-row:hover{border-color:var(--accent);}
.rec-row-top{display:flex;align-items:center;gap:.75rem;flex-wrap:wrap;}
.rec-rank{font-size:.75rem;color:var(--muted);font-weight:700;min-width:1.5rem;}
.rec-sym{font-weight:800;color:var(--accent);font-size:1.1rem;min-width:3.5rem;}
.rec-badge{font-size:.65rem;font-weight:700;padding:.2rem .5rem;border-radius:4px;white-space:nowrap;}
.rec-badge.rating-strong{background:rgba(102,187,106,.2);color:var(--success);}
.rec-badge.rating-good  {background:rgba(79,195,247,.15);color:var(--accent);}
.rec-badge.rating-mod   {background:rgba(255,167,38,.15);color:var(--warn);}
.rec-badge.rating-weak  {background:rgba(239,83,80,.12);color:var(--danger);}
.rating-strong{color:var(--success);}
.rating-good{color:var(--accent);}
.rating-mod{color:var(--warn);}
.rating-weak{color:var(--danger);}
.rec-score-bar-wrap{flex:1;background:var(--surface2);border-radius:999px;height:6px;overflow:hidden;min-width:60px;}
.rec-score-bar{height:100%;background:var(--accent);border-radius:999px;transition:width .4s;}
.rec-score-num{font-size:1rem;font-weight:800;white-space:nowrap;}
.rec-score-denom{font-size:.65rem;color:var(--muted);font-weight:400;}

.rec-row-prices{display:flex;align-items:center;gap:.5rem;flex-wrap:wrap;}
.rec-price-box{display:flex;flex-direction:column;align-items:center;background:var(--surface2);border:1px solid var(--border);border-radius:6px;padding:.4rem .75rem;min-width:70px;}
.rec-price-box.rec-price-buy   {border-color:var(--accent);background:rgba(79,195,247,.06);}
.rec-price-box.rec-price-stop  {border-color:var(--danger);background:rgba(239,83,80,.06);}
.rec-price-box.rec-price-target{border-color:var(--success);background:rgba(102,187,106,.06);}
.rec-price-lbl{font-size:.6rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);font-weight:600;}
.rec-price-val{font-size:1rem;font-weight:800;line-height:1.2;}
.rec-price-buy    .rec-price-val{color:var(--accent);}
.rec-price-stop   .rec-price-val{color:var(--danger);}
.rec-price-target .rec-price-val{color:var(--success);}
.rec-price-sub{font-size:.68rem;color:var(--muted);}
.rec-price-arrow{color:var(--muted);font-size:.8rem;padding:0 .1rem;}
.rec-price-divider{width:1px;height:36px;background:var(--border);margin:0 .25rem;}

.rec-breakdown{font-size:.72rem;color:var(--muted);}
.rec-breakdown b{color:var(--text);}

/* Rec card */
.rec-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.5rem;margin-bottom:1.25rem;}
.rec-header{display:flex;align-items:baseline;gap:.75rem;margin-bottom:1.25rem;}
.rec-ticker{font-size:1.4rem;font-weight:800;color:var(--accent);}
.rec-price{font-size:1.8rem;font-weight:700;}
.rec-currency{color:var(--muted);font-size:.85rem;}
.rec-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem;}
.rec-item{background:var(--surface2);border:1px solid var(--border);border-radius:8px;padding:1rem 1.25rem;border-top:3px solid;}
.rec-item.entry{border-top-color:var(--accent);}
.rec-item.stop{border-top-color:var(--danger);}
.rec-item.target{border-top-color:var(--success);}
.rec-label{font-size:.7rem;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);font-weight:600;margin-bottom:.4rem;}
.rec-value{font-size:1.5rem;font-weight:700;margin-bottom:.2rem;}
.rec-item.stop .rec-value{color:var(--danger);}
.rec-item.target .rec-value{color:var(--success);}
.rec-note{font-size:.75rem;color:var(--muted);}

.stats-row{display:grid;grid-template-columns:repeat(4,1fr);gap:.75rem;margin-bottom:1.25rem;}
.stat-card{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:.85rem 1rem;}
.stat-label{font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);margin-bottom:.3rem;font-weight:600;}
.stat-value{font-size:1.1rem;font-weight:700;}

.rules-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1rem;margin-bottom:1.25rem;}
.rule-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.25rem;border-left:4px solid;}
.rule-3{border-left-color:var(--warn);}
.rule-5{border-left-color:var(--accent);}
.rule-7{border-left-color:var(--success);}
.rule-num{font-size:2rem;font-weight:800;line-height:1;margin-bottom:.1rem;}
.rule-3 .rule-num{color:var(--warn);}
.rule-5 .rule-num{color:var(--accent);}
.rule-7 .rule-num{color:var(--success);}
.rule-title{font-size:.78rem;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);margin-bottom:1rem;font-weight:600;}
.rule-items{display:flex;flex-direction:column;gap:.5rem;}
.rule-row{display:flex;justify-content:space-between;font-size:.85rem;}
.rule-row span:first-child{color:var(--muted);}
.rule-row .val{font-weight:600;}

.model-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.5rem;margin-bottom:1.25rem;}
.section-title{font-size:.8rem;text-transform:uppercase;letter-spacing:.08em;color:var(--muted);font-weight:700;margin-bottom:1rem;}
.model-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:1rem;}
.model-item{background:var(--surface2);border:1px solid var(--border);border-radius:8px;padding:.9rem 1rem;}
.model-label{font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);margin-bottom:.35rem;font-weight:600;}
.model-value{font-size:1.2rem;font-weight:700;margin-bottom:.2rem;}
.model-value.pos{color:var(--success);}
.model-value.neg{color:var(--danger);}
.model-note{font-size:.72rem;color:var(--muted);}

.chart-card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1.5rem;margin-bottom:1.25rem;}
.chart-wrap{height:280px;margin-bottom:1rem;position:relative;}
.volume-wrap{height:100px;margin-bottom:0;}

.actions-row{display:flex;gap:.75rem;align-items:center;flex-wrap:wrap;margin-bottom:2rem;}
.btn-secondary{background:var(--surface2);color:var(--text);border:1px solid var(--border);padding:.55rem 1.2rem;border-radius:6px;font-size:.9rem;font-weight:600;cursor:pointer;transition:border-color .15s;}
.btn-secondary:hover:not(:disabled){border-color:var(--accent);color:var(--accent);}
.btn-secondary:disabled{opacity:.4;cursor:not-allowed;}
.action-msg{font-size:.85rem;color:var(--muted);}

/* Cal page */
.cal-desc{color:var(--muted);font-size:.88rem;line-height:1.6;}
.cal-banner{display:flex;align-items:center;gap:0;background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:1rem 1.5rem;margin-bottom:1.25rem;flex-wrap:wrap;gap:.5rem;}
.cal-banner-item{display:flex;flex-direction:column;align-items:center;padding:0 1.25rem;}
.cal-banner-lbl{font-size:.65rem;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);font-weight:600;margin-bottom:.25rem;}
.cal-banner-val{font-size:1.25rem;font-weight:700;}
.cal-banner-val.accent{color:var(--accent);}
.cal-banner-divider{width:1px;height:40px;background:var(--border);}

@media(max-width:680px){
  .rec-grid,.rules-grid{grid-template-columns:1fr;}
  .stats-row{grid-template-columns:repeat(2,1fr);}
  .model-grid{grid-template-columns:repeat(2,1fr);}
  .cal-banner{flex-direction:column;align-items:flex-start;}
  .cal-banner-divider{width:100%;height:1px;}
}
</style>
