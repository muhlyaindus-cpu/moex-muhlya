<template>
  <div class="dashboard">
    <div class="header">
      <h1>🚀 MOEX Trading Terminal</h1>
      <div class="status">
        <span class="dot" :class="{ green: connected }"></span>
        {{ connected ? 'Connected' : 'Connecting...' }}
      </div>
    </div>

    <!-- Панель выбора счёта -->
    <div class="account-panel">
      <h3>💰 Trading Account</h3>
      <div class="account-selector">
        <button 
          v-for="acc in accounts" 
          :key="acc.ID"
          :class="{ active: currentAccount === acc.ID }"
          @click="switchAccount(acc.ID)"
        >
          {{ acc.Name }} ({{ acc.Type }})
        </button>
      </div>
      <div class="current-account">
        Current: <strong>{{ formatAccountDisplay(currentAccount) || 'None' }}</strong>
      </div>
      <div class="current-account" v-if="tradingCode">
        Trading code: <strong>{{ tradingCode }}</strong>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <h3>Cash Balance</h3>
        <div class="value">{{ formatMoney(stats.cash_balance) }}</div>
        <div v-if="cashBalanceError" class="balance-warning">
          {{ cashBalanceError }}
          <br>
          Account: {{ formatAccountDisplay(currentAccount) || stats.account_id || 'unknown' }}
        </div>
      </div>
      <div class="stat-card">
        <h3>Portfolio Value</h3>
        <div class="value">{{ formatMoney(stats.total_value) }}</div>
      </div>
      <div class="stat-card">
        <h3>Instruments Value</h3>
        <div class="value">{{ formatMoney((stats.total_value || 0) - (stats.cash_balance || 0)) }}</div>
      </div>
      <div class="stat-card">
        <h3>Total P&L</h3>
        <div class="value" :class="{ positive: stats.total_pnl > 0, negative: stats.total_pnl < 0 }">
          {{ formatMoney(stats.total_pnl) }}
        </div>
      </div>
      <div class="stat-card">
        <h3>Positions</h3>
        <div class="value">{{ stats.positions_count }}</div>
      </div>
    </div>
    <div class="stats-freshness">
      <span class="badge" :class="{ cached: stats.stats_cached, live: !stats.stats_cached }">
        {{ stats.stats_cached ? 'cached' : 'live' }}
      </span>
      <span>updated: {{ formatStatsUpdatedAt(stats.stats_updated_at) }}</span>
    </div>
    <div class="session-grid">
      <div class="session-item">
        <span class="session-label">MOEX time (NSK):</span>
        <span class="session-time">{{ formatMarketClock(stats.market_time_local || '') }}</span>
      </div>
      <div class="session-item">
        <span class="session-label">Stocks:</span>
        <span class="session-pill" :class="{ open: stats.stock_session_open, closed: !stats.stock_session_open }">
          {{ stats.stock_session_open ? 'open' : 'closed' }}
        </span>
        <span v-if="!stats.stock_session_open" class="session-note">{{ stats.stock_session_note }}</span>
        <span v-if="!stats.stock_session_open && stockOpensInText" class="session-note">{{ stockOpensInText }}</span>
      </div>
      <div class="session-item">
        <span class="session-label">Futures:</span>
        <span class="session-pill" :class="{ open: stats.futures_session_open, closed: !stats.futures_session_open }">
          {{ stats.futures_session_open ? 'open' : 'closed' }}
        </span>
        <span v-if="!stats.futures_session_open" class="session-note">{{ stats.futures_session_note }}</span>
        <span v-if="!stats.futures_session_open && futuresOpensInText" class="session-note">{{ futuresOpensInText }}</span>
      </div>
    </div>

    <div class="agent-panel">
      <div class="panel-header">
        <h2>🤖 Мухля — торговый ассистент MOEX</h2>
        <div class="agent-controls">
          <button class="auto-btn" :class="{ active: autoTradingEnabled }" @click="toggleAutoTrading">
            {{ autoTradingEnabled ? 'Автоторговля: ON' : 'Разрешить автоторговлю' }}
          </button>
          <button class="toggle-btn" :class="{ active: agentActive }" @click="toggleAgent">
            {{ agentActive ? 'Active' : 'Inactive' }}
          </button>
        </div>
      </div>
      <div class="mode-status" :class="`kind-${tradingModeStatus.kind}`">
        {{ tradingModeStatus.text }}
      </div>
      <div class="market-status" :class="`kind-${marketSessionStatus.kind}`">
        {{ marketSessionStatus.text }}
      </div>
      <div v-if="marketSessionStatus.hint" class="market-hint">
        {{ marketSessionStatus.hint }}
      </div>
      <div class="silence-status" :class="`kind-${silenceReason.kind}`">
        {{ silenceReason.text }}
      </div>

      <div class="auto-instrument-config">
        <label>
          Режим выбора инструмента:
          <select v-model="autoInstrumentMode">
            <option value="auto">Все инструменты (акции + фьючерсы)</option>
            <option value="stocks">Только акции</option>
            <option value="futures">Только фьючерсы</option>
            <option value="options">Только опционы (скоро)</option>
            <option value="fixed">Только указанный инструмент</option>
          </select>
        </label>
        <label>
          Инструмент:
          <select v-model="autoFixedTicker" :disabled="autoInstrumentMode !== 'fixed'">
            <option value="" disabled>Выберите инструмент</option>
            <option v-for="item in filteredAnalyzeCandidates" :key="`auto-${item.ticker}`" :value="item.ticker">
              {{ item.ticker }} — {{ item.name }}
            </option>
          </select>
        </label>
        <button class="save-auto-config-btn" @click="saveAutoInstrumentConfig">Применить</button>
      </div>
      <div class="auto-instrument-config">
        <label>
          Бюджет Мухли в auto (% от кэша):
          <select v-model.number="autoBudgetPercent">
            <option v-for="p in [10,20,30,40,50,60,70,80,90,100]" :key="`pct-${p}`" :value="p">{{ p }}%</option>
          </select>
        </label>
        <div class="safety-presets">
          <span class="safety-presets-label">Пресет TP / Profit Lock (локально):</span>
          <button type="button" class="save-auto-config-btn preset-chip" title="TP 1.2%, SL 0.8%, Profit Lock 10%/3%" @click="applySafetyPreset('standard')">
            Стандарт
          </button>
        </div>
        <label>
          TP (%):
          <input v-model.number="autoTPPct" type="number" min="0.1" max="30" step="0.1">
        </label>
        <label>
          SL (%):
          <input v-model.number="autoSLPct" type="number" min="0.1" max="30" step="0.1">
        </label>
        <label>
          Profit Lock Trigger (%):
          <input v-model.number="autoProfitLockTriggerPct" type="number" min="0.1" max="100" step="0.1">
        </label>
        <label>
          Profit Lock Trail (%):
          <input v-model.number="autoProfitLockTrailPct" type="number" min="0.1" max="50" step="0.1">
        </label>
        <label>
          Cooldown блокировки тикера (мин):
          <input v-model.number="badTickerCooldownMin" type="number" min="1" max="240">
        </label>
        <button class="save-auto-config-btn" @click="saveSafetyConfig">Применить лимиты</button>
        <button class="save-auto-config-btn reset-soft-btn" @click="resetAutoContextSoft">Мягкий сброс контекста</button>
      </div>
      <div class="analysis-result" v-if="autoDecision">
        <div class="signal-reasoning">
          <strong>Почему Мухля выбрал инструмент:</strong>
          {{ autoDecision.note }}
        </div>
        <div class="signal-confidence">
          Universe: {{ autoDecision.universe_size }},
          Passed: {{ autoDecision.passed_filters }},
          cooldown-skip: {{ autoDecision.blocked_by_cooldown }},
          qual-skip: {{ autoDecision.blocked_by_qualification }},
          rotation-skip: {{ autoDecision.blocked_by_rotation }},
          cash-skip: {{ autoDecision.blocked_by_cash }},
          liq-skip: {{ autoDecision.rejected_by_liquidity }},
          price-skip: {{ autoDecision.rejected_by_price }}
        </div>
        <div class="signal-confidence">
          Best buy: {{ autoDecision.best_buy_ticker || '-' }} ({{ autoDecision.best_buy_conf || '0.00' }}) |
          Best sell: {{ autoDecision.best_sell_ticker || '-' }} ({{ autoDecision.best_sell_conf || '0.00' }}) |
          Action: {{ autoDecision.action_taken || '-' }}
        </div>
      </div>
      
      <div v-if="!autoTradingEnabled" class="analyze-form">
        <input
          v-model="analyzeQuery"
          placeholder="Запрос инструмента (например: Сбер, Si, Brent)"
          maxlength="32"
        >
        <select v-model="analyzeTicker">
          <option value="" disabled>Выберите инструмент</option>
          <option v-for="item in filteredAnalyzeCandidates" :key="item.ticker" :value="item.ticker">
            {{ item.ticker }} — {{ item.name }}
          </option>
        </select>
        <button @click="analyze">Analyze</button>
      </div>
      <div v-else class="analyze-hidden-note">
        Анализ конкретного инструмента доступен в ручной торговле.
      </div>
      
      <div v-if="analysisResult" class="analysis-result">
        <div class="signal-action" :class="analysisResult.action">
          {{ analysisResult.action ? analysisResult.action.toUpperCase() : 'NO SIGNAL' }}
        </div>
        <div class="signal-confidence" v-if="analysisResult.confidence">
          Confidence: {{ (analysisResult.confidence * 100).toFixed(0) }}%
        </div>
        <div class="signal-reasoning" v-if="analysisResult.reasoning">
          {{ analysisResult.reasoning }}
        </div>
        <div class="signal-error" v-if="analysisResult.error">
          Error: {{ analysisResult.error }}
        </div>
      </div>
    </div>

    <div class="market-tabs">
      <button 
        class="tab-btn" 
        :class="{ active: activeMarket === 'stocks' }" 
        @click="activeMarket = 'stocks'"
      >
        📈 Акции
      </button>
      <button 
        class="tab-btn" 
        :class="{ active: activeMarket === 'futures' }" 
        @click="activeMarket = 'futures'"
      >
        📊 Фьючерсы (FORTS)
      </button>
    </div>

    <!-- Акции -->
    <div v-if="activeMarket === 'stocks'" class="watchlist">
      <div class="section-header">
        <h2>📊 Watchlist (Акции)</h2>
        <button class="collapse-btn" type="button" @click="stocksWatchlistExpanded = !stocksWatchlistExpanded">
          {{ stocksWatchlistExpanded ? '−' : '+' }}
        </button>
      </div>
      <div v-if="stocksWatchlistExpanded" class="watchlist-search">
        <input
          v-model="stocksQuery"
          placeholder="Поиск акций MOEX (тикер или название)"
          maxlength="64"
        >
      </div>
      <div v-if="stocksWatchlistExpanded && watchlist.length === 0" class="loading">Loading...</div>
      <table v-else-if="stocksWatchlistExpanded">
        <thead>
          <tr>
            <th>Ticker</th>
            <th>Name</th>
            <th>Price</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in watchlist" :key="item.ticker">
            <td class="ticker">{{ item.ticker }}</td>
            <td>{{ item.name }}</td>
            <td class="price">{{ formatMoney(item.price) }}</td>
            <td>
              <button class="buy" @click="quickBuy(item.ticker, item.price)">Buy</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Фьючерсы -->
    <div v-if="activeMarket === 'futures'" class="watchlist">
      <div class="section-header">
        <h2>📊 Watchlist (Фьючерсы FORTS)</h2>
        <button class="collapse-btn" type="button" @click="futuresWatchlistExpanded = !futuresWatchlistExpanded">
          {{ futuresWatchlistExpanded ? '−' : '+' }}
        </button>
      </div>
      <div v-if="futuresWatchlistExpanded" class="watchlist-search">
        <input
          v-model="futuresQuery"
          placeholder="Поиск фьючерсов FORTS (тикер или название)"
          maxlength="64"
        >
      </div>
      <div v-if="futuresWatchlistExpanded && futuresWatchlist.length === 0" class="loading">Loading...</div>
      <table v-else-if="futuresWatchlistExpanded">
        <thead>
          <tr>
            <th>Ticker</th>
            <th>Name</th>
            <th>Price</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in futuresWatchlist" :key="item.ticker">
            <td class="ticker">{{ item.ticker }}</td>
            <td>{{ item.name }}</td>
            <td class="price">{{ formatMoney(item.price) }}</td>
            <td>
              <button class="buy" @click="quickBuyFutures(item.ticker, item.price)">Buy</button>
              <button class="sell" @click="quickSellFutures(item.ticker, item.price)">Sell</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="portfolio">
      <h2>💼 Portfolio</h2>
      <div class="portfolio-toolbar">
        <label class="checkbox-inline">
          <input v-model="showZeroPositions" type="checkbox">
          Показывать нулевые позиции
        </label>
      </div>
      <div v-if="visiblePortfolio.length === 0" class="loading">No positions</div>
      <table v-else>
        <thead>
          <tr>
            <th>Ticker</th>
            <th>Quantity</th>
            <th>Avg Price</th>
            <th>Current Price</th>
            <th>P&L</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="pos in visiblePortfolio" :key="pos.ticker">
            <td class="ticker" :title="formatTickerMeta(pos.ticker)">
              <div>{{ formatTickerMain(pos.ticker) }}</div>
              <div class="ticker-meta" v-if="formatTickerMeta(pos.ticker)">{{ formatTickerMeta(pos.ticker) }}</div>
              <div class="ticker-issuer" v-if="formatIssuerName(pos.ticker)">{{ formatIssuerName(pos.ticker) }}</div>
            </td>
            <td>{{ pos.quantity }}</td>
            <td>{{ formatMoney(pos.avg_price) }}</td>
            <td>{{ formatMoney(pos.current_price) }}</td>
            <td class="pnl" :class="{ positive: pos.pnl > 0, negative: pos.pnl < 0 }">
              {{ formatMoney(pos.pnl) }}
            </td>
            <td>
              <button class="sltp" type="button" @click="placeSLTPBracket(pos)">SL/TP</button>
              <button 
                class="close-position" 
                :class="pos.quantity > 0 ? 'close-long' : 'close-short'"
                @click="closePosition(pos.ticker, pos.quantity, pos.current_price)"
              >
                {{ pos.quantity > 0 ? 'Закрыть Long' : 'Закрыть Short' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="portfolio">
      <div class="section-header">
        <h2>🧾 Журнал сделок</h2>
        <button class="collapse-btn" type="button" @click="tradeJournalExpanded = !tradeJournalExpanded">
          {{ tradeJournalExpanded ? '−' : '+' }}
        </button>
      </div>
      <div v-if="tradeJournalExpanded && tradeJournal.length === 0" class="loading">Сделок пока нет</div>
      <table v-else-if="tradeJournalExpanded">
        <thead>
          <tr>
            <th>Time</th>
            <th>Источник</th>
            <th>Ticker</th>
            <th>Side</th>
            <th>Qty</th>
            <th>Price</th>
            <th>Status</th>
            <th>P/L</th>
            <th>Reason</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="t in tradeJournal"
            :key="`${t.source || 'local'}-${t.external_id || t.id}`"
          >
            <td>{{ formatTime(t.executed_at) }}</td>
            <td>
              <span class="journal-source" :class="`src-${t.source || 'local'}`">
                {{ t.source === 'finam' ? 'Finam API' : 'Локально' }}
              </span>
            </td>
            <td class="ticker" :title="formatTickerMeta(t.ticker)">
              <div>{{ formatTickerMain(t.ticker) }}</div>
              <div class="ticker-meta" v-if="formatTickerMeta(t.ticker)">{{ formatTickerMeta(t.ticker) }}</div>
              <div class="ticker-issuer" v-if="formatIssuerName(t.ticker)">{{ formatIssuerName(t.ticker) }}</div>
            </td>
            <td :class="{ positive: t.side === 'buy', negative: t.side === 'sell' }">{{ t.side }}</td>
            <td>{{ t.quantity }}</td>
            <td>{{ formatMoney(t.price) }}</td>
            <td>
              <span class="trade-status" :class="`kind-${getTradeStatusKind(t.status)}`">
                {{ getTradeStatusLabel(t.status) }}
              </span>
            </td>
            <td class="pnl" :class="{ positive: (t.realized_pl || 0) > 0, negative: (t.realized_pl || 0) < 0 }">
              {{ formatRealizedPL(t.realized_pl) }}
            </td>
            <td class="trade-reason" :title="getTradeStatusReason(t.status)">
              {{ getTradeStatusReason(t.status) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="portfolio">
      <div class="section-header runtime-header">
        <h2>🛰️ Лента шагов программы</h2>
        <div class="runtime-controls">
          <label>
            Лимит:
            <select v-model.number="runtimeEventsLimit">
              <option :value="10">10</option>
              <option :value="50">50</option>
              <option :value="120">120</option>
            </select>
          </label>
          <label>
            Источник:
            <select v-model="runtimeEventsSource">
              <option value="db">Postgres</option>
              <option value="memory">RAM (текущий runtime)</option>
            </select>
          </label>
          <button class="collapse-btn runtime-refresh-btn" type="button" @click="loadRuntimeEvents">
            ↻
          </button>
        </div>
      </div>
      <div v-if="runtimeEvents.length === 0" class="loading">Событий пока нет</div>
      <table v-else>
        <thead>
          <tr>
            <th>Time</th>
            <th>Level</th>
            <th>Step</th>
            <th>Details</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(ev, idx) in runtimeEvents" :key="`${ev.at}-${ev.step}-${idx}`">
            <td>{{ formatEventTime(ev.at) }}</td>
            <td>
              <span class="trade-status" :class="`kind-${getEventKind(ev.level)}`">{{ ev.level }}</span>
            </td>
            <td>{{ ev.step }}</td>
            <td class="trade-reason" :title="ev.details">{{ ev.details }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="portfolio">
      <div class="section-header">
        <h2>🚫 Заблокированные бумаги</h2>
        <div class="runtime-controls">
          <button class="collapse-btn runtime-refresh-btn" type="button" @click="loadBlockedTickers">
            ↻
          </button>
        </div>
      </div>
      <div v-if="blockedTickers.length === 0" class="loading">Нет заблокированных бумаг</div>
      <table v-else>
        <thead>
          <tr>
            <th>Ticker</th>
            <th>Тип</th>
            <th>Причина блокировки</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in blockedTickers" :key="t.ticker">
            <td class="ticker">{{ t.ticker }}</td>
            <td>
              <span class="trade-status" :class="t.type === 'futures' ? 'kind-info' : 'kind-warn'">
                {{ t.type === 'futures' ? 'Фьючерс' : 'Акция' }}
              </span>
            </td>
            <td class="trade-reason" :title="t.reason">{{ t.reason }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import {
  parseClockToMinutes,
  formatMarketClock,
  formatDuration,
  secondsToNext,
  formatDateTimeNskMsk,
  formatClockNskMsk
} from '../utils/time'
import {
  GetWatchlist,
  GetStockInstruments,
  GetPortfolio,
  GetStatistics,
  GetAgentStatus,
  SetAgentActive,
  AnalyzeWithAgent,
  BuyPosition,
  SellPosition,
  GetAccounts,
  SwitchAccount,
  GetCurrentAccount,
  GetFuturesWatchlist,
  GetFuturesInstruments,
  BuyFutures,
  SellFutures,
  GetMode,
  SetMode,
  StartAutoTrading,
  StopAutoTrading,
  GetTradeJournal,
  PlaceSLTPBrackets,
  GetAutoInstrumentConfig,
  SetAutoInstrumentConfig,
  GetRuntimeEvents,
  GetRuntimeEventsPersistent,
  GetSafetyConfig,
  SetBadTickerCooldownMinutes,
  GetAutoDecisionStatus,
  SetAutoBudgetPercent,
  SetAutoTPSLPercent,
  SetAutoProfitLockPercent,
  ResetAutoContextSoft,
  GetBlockedTickers
} from '../../wailsjs/go/main/App'

interface Security {
  ticker: string
  name: string
  price: number
}

interface Position {
  ticker: string
  quantity: number
  avg_price: number
  current_price: number
  pnl: number
}

interface Statistics {
  cash_balance: number
  total_value: number
  total_pnl: number
  positions_count: number
  return_pct: number
  cash_balance_error?: string
  account_id?: string
  stats_cached?: boolean
  stats_updated_at?: string
  market_tz?: string
  stock_session_open?: boolean
  stock_session_note?: string
  futures_session_open?: boolean
  futures_session_note?: string
  market_time_local?: string
}

interface TradeJournalEntry {
  id: number
  ticker: string
  side: string
  quantity: number
  price: number
  status: string
  executed_at: string
  realized_pl?: number
  source?: string
  external_id?: string
}

interface RuntimeEvent {
  at: string
  level: string
  step: string
  details: string
}

interface BlockedTicker {
  ticker: string
  reason: string
  type: string
}

interface AutoDecisionStatus {
  at: string
  universe_size: number
  passed_filters: number
  blocked_by_cooldown: number
  blocked_by_qualification: number
  blocked_by_rotation: number
  blocked_by_cash: number
  rejected_by_liquidity: number
  rejected_by_price: number
  best_buy_ticker: string
  best_buy_conf: string
  best_sell_ticker: string
  best_sell_conf: string
  action_taken: string
  note: string
}

interface Account {
  ID: string
  Name: string
  Type: string
  TradingCode?: string
}

const connected = ref(true)
const watchlist = ref<Security[]>([])
const futuresWatchlist = ref<Security[]>([])
const portfolio = ref<Position[]>([])
const showZeroPositions = ref(false)
const agentActive = ref(false)
const analyzeQuery = ref('')
const analyzeTicker = ref('')
const analysisResult = ref<any>(null)
const activeMarket = ref('stocks')
const accounts = ref<Account[]>([])
const currentAccount = ref('')
const tradingCode = ref('')
const filteredAnalyzeCandidates = ref<Security[]>([])
const stocksQuery = ref('')
const futuresQuery = ref('')
const autoTradingEnabled = ref(false)
const tradeJournal = ref<TradeJournalEntry[]>([])
const runtimeEvents = ref<RuntimeEvent[]>([])
const runtimeEventsSource = ref<'db' | 'memory'>('db')
const runtimeEventsLimit = ref(50)
const blockedTickers = ref<BlockedTicker[]>([])
const autoDecision = ref<AutoDecisionStatus | null>(null)
const issuerCache = ref<Record<string, string>>({})
const autoInstrumentMode = ref<'auto' | 'stocks' | 'futures' | 'options' | 'fixed'>('auto')
const autoFixedTicker = ref('')
const badTickerCooldownMin = ref(30)
const autoBudgetPercent = ref(100)
const autoTPPct = ref(1.2)
const autoSLPct = ref(0.8)
const autoProfitLockTriggerPct = ref(10.0)
const autoProfitLockTrailPct = ref(3.0)
const stocksWatchlistExpanded = ref(false)
const futuresWatchlistExpanded = ref(false)
const tradeJournalExpanded = ref(false)
const stocksCollapseStorageKey = 'trading.watchlist.stocks.expanded'
const futuresCollapseStorageKey = 'trading.watchlist.futures.expanded'
const tradeJournalCollapseStorageKey = 'trading.tradejournal.expanded'

const stats = ref<Statistics>({
  cash_balance: 0,
  total_value: 0,
  total_pnl: 0,
  positions_count: 0,
  return_pct: 0,
  cash_balance_error: '',
  account_id: '',
  stats_cached: false,
  stats_updated_at: '',
  market_tz: 'Asia/Novosibirsk',
  stock_session_open: false,
  stock_session_note: '',
  futures_session_open: false,
  futures_session_note: '',
  market_time_local: ''
})

const cashBalanceError = ref('')
const hasClearing = (note?: string): boolean => (note || '').toLowerCase().includes('clearing')
const sessionOpensInText = (isOpen: boolean, openHour: number, openMinute: number): string => {
  if (isOpen) return ''
  const mins = parseClockToMinutes(stats.value.market_time_local || '')
  const left = formatDuration(secondsToNext(mins, openHour, openMinute))
  return `Откроется через ${left}.`
}
const stockOpensInText = computed(() => sessionOpensInText(!!stats.value.stock_session_open, 6, 50))
const futuresOpensInText = computed(() => sessionOpensInText(!!stats.value.futures_session_open, 8, 50))

const tradingModeStatus = computed(() => {
  if (autoTradingEnabled.value) {
    const stockClearing = hasClearing(stats.value.stock_session_note)
    const futClearing = hasClearing(stats.value.futures_session_note)
    if (!stats.value.stock_session_open && !stats.value.futures_session_open) {
      if (stockClearing || futClearing) {
        return {
          text: 'Режим: AUTO (рынки в клиринге/переходе, ждём возобновление)',
          kind: 'analysis'
        }
      }
      return {
        text: 'Режим: AUTO (ожидание сессии)',
        kind: 'analysis'
      }
    }
    if (!stats.value.stock_session_open && stats.value.futures_session_open && stockClearing) {
      return {
        text: 'Режим: AUTO (акции в клиринге/переходе, торгуем срочный)',
        kind: 'analysis'
      }
    }
    if (stats.value.stock_session_open && !stats.value.futures_session_open && futClearing) {
      return {
        text: 'Режим: AUTO (срочный в клиринге, торгуем фондовый)',
        kind: 'analysis'
      }
    }
    if (autoInstrumentMode.value === 'fixed') {
      const ticker = (autoFixedTicker.value || '').trim().toUpperCase()
      if (ticker) {
        return {
          text: `Режим: торгует агент (фиксированный инструмент: ${ticker})`,
          kind: 'agent'
        }
      }
      return {
        text: 'Режим: торгует агент (фиксированный инструмент не выбран)',
        kind: 'agent'
      }
    }
    if (autoInstrumentMode.value === 'stocks') {
      return {
        text: 'Режим: торгует агент (только акции)',
        kind: 'agent'
      }
    }
    if (autoInstrumentMode.value === 'futures') {
      return {
        text: 'Режим: торгует агент (только фьючерсы)',
        kind: 'agent'
      }
    }
    if (autoInstrumentMode.value === 'options') {
      return {
        text: 'Режим: торгует агент (только опционы, пул пока пуст)',
        kind: 'analysis'
      }
    }
    return {
      text: 'Режим: торгует агент (сам выбирает любые инструменты)',
      kind: 'agent'
    }
  }
  if (agentActive.value) {
    return {
      text: 'Режим: ручная торговля (агент только анализирует)',
      kind: 'analysis'
    }
  }
  return {
    text: 'Режим: ручная торговля (торгует пользователь)',
    kind: 'manual'
  }
})

const marketSessionStatus = computed(() => {
  const stockOpen = !!stats.value.stock_session_open
  const futOpen = !!stats.value.futures_session_open
  const stockClearing = hasClearing(stats.value.stock_session_note)
  const futClearing = hasClearing(stats.value.futures_session_note)
  const mins = parseClockToMinutes(stats.value.market_time_local || '')

  if (stockOpen && futOpen) {
    return {
      text: 'Рынок: фондовый открыт / срочный открыт',
      kind: 'open',
      hint: ''
    }
  }
  if (!stockOpen && futOpen && stockClearing) {
    const left = formatDuration(secondsToNext(mins, 19, 1))
    return {
      text: 'Рынок: фондовый клиринг/переход / срочный открыт',
      kind: 'clearing',
      hint: `Акции возобновятся после перехода сессии (ориентир 19:01 NSK), через ${left}.`
    }
  }
  if (stockOpen && !futOpen && futClearing) {
    const left = formatDuration(secondsToNext(mins, 0, 30))
    return {
      text: 'Рынок: фондовый открыт / срочный клиринг',
      kind: 'clearing',
      hint: `Срочный рынок возобновится после клиринга около 00:30 NSK (20:30 MSK), через ${left}.`
    }
  }
  if (stockOpen && !futOpen) {
    const left = formatDuration(secondsToNext(mins, 8, 50))
    return {
      text: 'Рынок: фондовый открыт / срочный закрыт',
      kind: 'partial',
      hint: `Срочный рынок откроется в 08:50 NSK (04:50 MSK), через ${left}.`
    }
  }
  if (!stockOpen && futOpen) {
    const left = formatDuration(secondsToNext(mins, 6, 50))
    return {
      text: 'Рынок: фондовый закрыт / срочный открыт',
      kind: 'partial',
      hint: `Фондовый рынок откроется в 06:50 NSK (02:50 MSK), через ${left}.`
    }
  }
  const weekday = new Date().getDay()
  const weekendHint = weekday === 0 || weekday === 6
  const leftStock = formatDuration(secondsToNext(mins, 6, 50))
  const leftFut = formatDuration(secondsToNext(mins, 8, 50))
  return {
    text: 'Рынок: фондовый закрыт / срочный закрыт',
    kind: 'closed',
    hint: weekendHint
      ? 'Сегодня, вероятно, выходной/праздничный режим. Проверь торговый календарь MOEX.'
      : `Ближайшее открытие: фондовый 06:50 NSK (02:50 MSK, через ${leftStock}), срочный 08:50 NSK (04:50 MSK, через ${leftFut}).`
  }
})

const silenceReason = computed(() => {
  if (!autoTradingEnabled.value) {
    return { kind: 'neutral', text: 'Мухля молчит: автоторговля выключена.' }
  }
  if (!agentActive.value) {
    return { kind: 'neutral', text: 'Мухля молчит: агент не активен.' }
  }
  if (!stats.value.stock_session_open && !stats.value.futures_session_open) {
    return { kind: 'warn', text: 'Мухля молчит: оба рынка сейчас закрыты.' }
  }
  const events = runtimeEvents.value || []
  const pauseEvent = events.find((e) => e.step === 'auto.pause_loss_streak')
  if (pauseEvent) {
    return { kind: 'risk', text: `Мухля на паузе после убытков: ${pauseEvent.details}` }
  }
  if ((autoDecision.value?.universe_size || 0) === 0) {
    return { kind: 'warn', text: 'Мухля молчит: нет доступных инструментов в universe (0).' }
  }
  if ((autoDecision.value?.passed_filters || 0) === 0) {
    return { kind: 'warn', text: 'Мухля молчит: кандидаты отсеяны фильтрами (Passed=0).' }
  }
  const hasSkips =
    (autoDecision.value?.blocked_by_cooldown || 0) > 0 ||
    (autoDecision.value?.blocked_by_qualification || 0) > 0 ||
    (autoDecision.value?.blocked_by_cash || 0) > 0 ||
    (autoDecision.value?.rejected_by_liquidity || 0) > 0 ||
    (autoDecision.value?.rejected_by_price || 0) > 0
  if (hasSkips) {
    return { kind: 'warn', text: 'Мухля молчит: сигнал есть, но вход блокируют фильтры/ограничения.' }
  }
  if ((autoDecision.value?.action_taken || '') === 'no_order') {
    return { kind: 'ok', text: 'Мухля наблюдает рынок: кандидаты есть, но условия входа пока не подтверждены.' }
  }
  return { kind: 'ok', text: 'Мухля в рабочем цикле.' }
})

async function loadAccounts() {
  try {
    accounts.value = await GetAccounts()
    currentAccount.value = await GetCurrentAccount()
    tradingCode.value = accounts.value[0]?.TradingCode || ''
  } catch (e) {
    console.error('Error loading accounts:', e)
  }
}

async function switchAccount(accountName: string) {
  try {
    const success = await SwitchAccount(accountName)
    if (success) {
      currentAccount.value = accountName
      loadData()
    }
  } catch (e) {
    console.error('Error switching account:', e)
  }
}

async function loadData() {
  try {
    watchlist.value = stocksQuery.value.trim()
      ? await GetStockInstruments(stocksQuery.value.trim(), 120)
      : await GetWatchlist()
    futuresWatchlist.value = futuresQuery.value.trim()
      ? await GetFuturesInstruments(futuresQuery.value.trim(), 120)
      : await GetFuturesWatchlist()
    updateAnalyzeCandidates()
    portfolio.value = await GetPortfolio()
    const data = await GetStatistics()
    stats.value = {
      cash_balance: data.cash_balance || 0,
      total_value: data.total_value || 0,
      total_pnl: data.total_pnl || 0,
      positions_count: data.positions_count || 0,
      return_pct: data.return_pct || 0,
      cash_balance_error: data.cash_balance_error || '',
      account_id: data.account_id || '',
      stats_cached: !!data.stats_cached,
      stats_updated_at: data.stats_updated_at || '',
      market_tz: data.market_tz || 'Asia/Novosibirsk',
      stock_session_open: !!data.stock_session_open,
      stock_session_note: data.stock_session_note || '',
      futures_session_open: !!data.futures_session_open,
      futures_session_note: data.futures_session_note || '',
      market_time_local: data.market_time_local || ''
    }
    cashBalanceError.value = data.cash_balance_error || ''
    const status = await GetAgentStatus()
    agentActive.value = status.active
    const mode = await GetMode()
    autoTradingEnabled.value = mode === 'auto'
    const autoCfg = await GetAutoInstrumentConfig()
    const cfgMode = (
      autoCfg.mode === 'fixed' ||
      autoCfg.mode === 'stocks' ||
      autoCfg.mode === 'futures' ||
      autoCfg.mode === 'options'
    ) ? autoCfg.mode : 'auto'
    autoInstrumentMode.value = cfgMode
    autoFixedTicker.value = cfgMode === 'fixed' ? (autoCfg.ticker || '') : ''
    const safetyCfg = await GetSafetyConfig()
    badTickerCooldownMin.value = Math.max(1, Number(safetyCfg?.bad_ticker_cooldown_min || 30))
    autoBudgetPercent.value = Math.max(10, Math.min(100, Number(safetyCfg?.auto_budget_percent || 100)))
    autoTPPct.value = Math.max(0.1, Number(safetyCfg?.auto_tp_pct || 1.2))
    autoSLPct.value = Math.max(0.1, Number(safetyCfg?.auto_sl_pct || 0.8))
    autoProfitLockTriggerPct.value = Math.max(0.1, Number(safetyCfg?.auto_profit_lock_trigger_pct || 10.0))
    autoProfitLockTrailPct.value = Math.max(0.1, Number(safetyCfg?.auto_profit_lock_trail_pct || 3.0))
    tradeJournal.value = await GetTradeJournal(80)
    await loadRuntimeEvents()
    autoDecision.value = await GetAutoDecisionStatus()
    await enrichIssuerCacheFromKnownTickers()
  } catch (e) {
    console.error('Error loading data:', e)
    connected.value = false
  }
}

async function loadRuntimeEvents() {
  const limit = Math.max(10, Math.min(120, Number(runtimeEventsLimit.value || 50)))
  if (runtimeEventsSource.value === 'db') {
    runtimeEvents.value = await GetRuntimeEventsPersistent(limit)
    return
  }
  runtimeEvents.value = await GetRuntimeEvents(limit)
}

async function loadBlockedTickers() {
  try {
    blockedTickers.value = await GetBlockedTickers()
  } catch (e) {
    console.error('Failed to load blocked tickers:', e)
    blockedTickers.value = []
  }
}

function loadWatchlistCollapseState() {
  try {
    const stocksSaved = localStorage.getItem(stocksCollapseStorageKey)
    const futuresSaved = localStorage.getItem(futuresCollapseStorageKey)
    const tradeJournalSaved = localStorage.getItem(tradeJournalCollapseStorageKey)
    if (stocksSaved !== null) {
      stocksWatchlistExpanded.value = stocksSaved === '1'
    }
    if (futuresSaved !== null) {
      futuresWatchlistExpanded.value = futuresSaved === '1'
    }
    if (tradeJournalSaved !== null) {
      tradeJournalExpanded.value = tradeJournalSaved === '1'
    }
  } catch (e) {
    console.warn('Cannot load watchlist collapsed state:', e)
  }
}

let queryDebounceTimer: ReturnType<typeof setTimeout> | null = null

function onWatchlistQueryChanged() {
  if (queryDebounceTimer) {
    clearTimeout(queryDebounceTimer)
  }
  queryDebounceTimer = setTimeout(() => {
    loadData()
  }, 350)
}

function updateAnalyzeCandidates() {
  const query = analyzeQuery.value.trim().toLowerCase()
  const merged = [...watchlist.value, ...futuresWatchlist.value]
  const uniq = new Map<string, Security>()
  for (const s of merged) {
    if (!uniq.has(s.ticker)) {
      uniq.set(s.ticker, s)
    }
  }

  let result = Array.from(uniq.values())
  if (query) {
    result = result.filter((s) =>
      s.ticker.toLowerCase().includes(query) || s.name.toLowerCase().includes(query),
    )
  }
  filteredAnalyzeCandidates.value = result

  if (!analyzeTicker.value && result.length > 0) {
    analyzeTicker.value = result[0].ticker
  }
}

async function quickBuy(ticker: string, price: number) {
  const quantity = prompt('Quantity:', '10')
  if (quantity) {
    const success = await BuyPosition(ticker, parseInt(quantity), price)
    if (success) {
      loadData()
    } else {
      alert('Buy failed')
    }
  }
}

async function placeSLTPBracket(pos: Position) {
  const defSide = pos.quantity > 0 ? 'sell' : 'buy'
  const closeSide = prompt(
    'Сторона закрытия позиции: buy или sell (для лонга обычно sell):',
    defSide
  )
  if (closeSide === null || !closeSide.trim()) {
    return
  }
  const slStr = prompt('Stop-loss: цена (0 = не ставить):', '0')
  const tpStr = prompt('Take-profit: цена (0 = не ставить):', '0')
  if (slStr === null || tpStr === null) {
    return
  }
  const sl = parseFloat(String(slStr).replace(',', '.'))
  const tp = parseFloat(String(tpStr).replace(',', '.'))
  if ((Number.isNaN(sl) || sl <= 0) && (Number.isNaN(tp) || tp <= 0)) {
    alert('Укажите хотя бы одну цену SL или TP больше нуля.')
    return
  }
  const qty = Math.abs(pos.quantity)
  const slQty = !Number.isNaN(sl) && sl > 0 ? qty : 0
  const slPrice = !Number.isNaN(sl) && sl > 0 ? sl : 0
  const tpQty = !Number.isNaN(tp) && tp > 0 ? qty : 0
  const tpPrice = !Number.isNaN(tp) && tp > 0 ? tp : 0
  try {
    const id = await PlaceSLTPBrackets(pos.ticker, closeSide.trim(), slQty, slPrice, tpQty, tpPrice)
    alert(`SL/TP заявка принята. order_id: ${id}`)
    await loadData()
  } catch (e: unknown) {
    alert(e instanceof Error ? e.message : String(e))
  }
}

async function quickSell(ticker: string, maxQuantity: number, price: number) {
  const quantity = prompt(`Quantity (max: ${maxQuantity}):`, maxQuantity.toString())
  if (quantity && parseInt(quantity) <= maxQuantity) {
    const success = await SellPosition(ticker, parseInt(quantity), price)
    if (success) {
      loadData()
    } else {
      alert('Sell failed')
    }
  }
}

// closePosition закрывает позицию (long или short)
async function closePosition(ticker: string, quantity: number, price: number) {
  const absQuantity = Math.abs(quantity)
  const isLong = quantity > 0
  
  const confirmMsg = isLong 
    ? `Закрыть LONG ${ticker} (${absQuantity} шт) по цене ${price}?`
    : `Закрыть SHORT ${ticker} (${absQuantity} шт) по цене ${price}?`
  
  if (!confirm(confirmMsg)) {
    return
  }
  
  let success = false
  if (isLong) {
    // Закрытие long — продаём
    success = await SellPosition(ticker, absQuantity, price)
  } else {
    // Закрытие short — покупаем
    success = await BuyPosition(ticker, absQuantity, price)
  }
  
  if (success) {
    loadData()
  } else {
    alert(isLong ? 'Failed to close long position' : 'Failed to close short position')
  }
}

async function quickBuyFutures(ticker: string, price: number) {
  const quantity = prompt('Quantity (contracts):', '1')
  if (quantity) {
    const success = await BuyFutures(ticker, parseInt(quantity), price)
    if (success) {
      loadData()
    } else {
      alert('Buy futures failed')
    }
  }
}

async function quickSellFutures(ticker: string, price: number) {
  const quantity = prompt('Quantity (contracts):', '1')
  if (quantity) {
    const success = await SellFutures(ticker, parseInt(quantity), price)
    if (success) {
      loadData()
    } else {
      alert('Sell futures failed')
    }
  }
}

async function toggleAgent() {
  agentActive.value = await SetAgentActive(!agentActive.value)
}

async function toggleAutoTrading() {
  try {
    if (!autoTradingEnabled.value) {
      await SetAgentActive(true)
      await SetMode('auto')
      const started = await StartAutoTrading()
      if (!started) {
        alert('Не удалось запустить автоторговлю')
        return
      }
      autoTradingEnabled.value = true
      agentActive.value = true
      return
    }

    const stopped = await StopAutoTrading()
    if (!stopped) {
      alert('Не удалось остановить автоторговлю')
      return
    }
    await SetMode('manual')
    autoTradingEnabled.value = false
  } catch (e) {
    console.error('Error toggling auto trading:', e)
    alert('Ошибка переключения автоторговли')
  }
}

async function saveAutoInstrumentConfig() {
  try {
    const ticker =
      autoInstrumentMode.value === 'fixed'
        ? (autoFixedTicker.value || '').trim().toUpperCase()
        : ''
    if (autoInstrumentMode.value === 'fixed' && !ticker) {
      alert('Выберите инструмент для фиксированного режима.')
      return
    }
    const ok = await SetAutoInstrumentConfig(autoInstrumentMode.value, ticker)
    if (!ok) {
      alert('Не удалось применить настройку выбора инструмента.')
      return
    }
    autoFixedTicker.value = ticker
    await loadData()
  } catch (e) {
    console.error('Error saving auto instrument config:', e)
    alert('Ошибка сохранения настройки инструмента.')
  }
}

/** Локальные пресеты: не трогают VPS — только текущий runtime через Set* при сохранении. */
const SAFETY_PRESETS = {
  standard: { tp: 1.2, sl: 0.8, plTrigger: 10.0, plTrail: 3.0 }
} as const

async function applySafetyPreset(key: keyof typeof SAFETY_PRESETS) {
  const p = SAFETY_PRESETS[key]
  autoTPPct.value = p.tp
  autoSLPct.value = p.sl
  autoProfitLockTriggerPct.value = p.plTrigger
  autoProfitLockTrailPct.value = p.plTrail
  await saveSafetyConfig()
}

async function saveSafetyConfig() {
  try {
    const min = Math.max(1, Math.floor(Number(badTickerCooldownMin.value || 30)))
    const ok = await SetBadTickerCooldownMinutes(min)
    const pct = Math.max(10, Math.min(100, Math.floor(Number(autoBudgetPercent.value || 100) / 10) * 10))
    const okPct = await SetAutoBudgetPercent(pct)
    const tp = Math.max(0.1, Number(autoTPPct.value || 1.2))
    const sl = Math.max(0.1, Number(autoSLPct.value || 0.8))
    const okTPSL = await SetAutoTPSLPercent(tp, sl)
    const plTrigger = Math.max(0.1, Number(autoProfitLockTriggerPct.value || 10.0))
    const plTrail = Math.max(0.1, Number(autoProfitLockTrailPct.value || 3.0))
    const okPL = await SetAutoProfitLockPercent(plTrigger, plTrail)
    if (!ok || !okPct || !okTPSL || !okPL) {
      alert('Не удалось применить лимиты безопасности.')
      return
    }
    badTickerCooldownMin.value = min
    autoBudgetPercent.value = pct
    autoTPPct.value = tp
    autoSLPct.value = sl
    autoProfitLockTriggerPct.value = plTrigger
    autoProfitLockTrailPct.value = plTrail
    await loadData()
  } catch (e) {
    console.error('Error saving safety config:', e)
    alert('Ошибка сохранения safety-настройки.')
  }
}

async function resetAutoContextSoft() {
  try {
    const ok = await ResetAutoContextSoft()
    if (!ok) {
      alert('Не удалось сбросить авто-контекст.')
      return
    }
    await loadData()
  } catch (e) {
    console.error('Error resetting auto context softly:', e)
    alert('Ошибка мягкого сброса контекста.')
  }
}

async function analyze() {
  if (!analyzeTicker.value) return
  const result = await AnalyzeWithAgent(analyzeTicker.value.toUpperCase())
  analysisResult.value = result
}

function formatMoney(value: number): string {
  if (!value && value !== 0) return '0 ₽'
  return value.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ' ') + ' ₽'
}

function formatAccountDisplay(accountId: string): string {
  if (!accountId) return ''
  if (accountId.toUpperCase().startsWith('КЛФ-')) return accountId
  if (/^\d+$/.test(accountId)) return `КлФ-${accountId}`
  return accountId
}

function formatTime(ts: string): string {
  return formatDateTimeNskMsk(ts)
}

function formatRealizedPL(value?: number): string {
  if (value === undefined || value === null) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${formatMoney(value)}`
}

function formatStatsUpdatedAt(ts?: string): string {
  return formatClockNskMsk(ts)
}

function formatEventTime(ts: string): string {
  return formatTime(ts)
}

function isLikelyFuturesTicker(ticker: string): boolean {
  return /[FGHJKMNQUVXZ]\d$/i.test(ticker)
}

function formatTickerVariants(rawTicker: string): string[] {
  const raw = (rawTicker || '').trim().toUpperCase()
  if (!raw) return []
  const out: string[] = [raw]
  if (raw.includes('@') || raw.includes('_')) {
    return out
  }
  if (isLikelyFuturesTicker(raw)) {
    out.push(`${raw}_SPBFUT`)
    out.push(`${raw}@MISX`)
    return out
  }
  out.push(`${raw}_TQBR`)
  out.push(`${raw}@MISX`)
  return out
}

function formatTickerMain(rawTicker: string): string {
  const variants = formatTickerVariants(rawTicker)
  return variants[0] || '-'
}

function formatTickerMeta(rawTicker: string): string {
  const variants = formatTickerVariants(rawTicker)
  if (variants.length <= 1) return ''
  return variants.slice(1).join(' | ')
}

const issuerByTicker = computed(() => {
  const m = new Map<string, string>()
  for (const [t, n] of Object.entries(issuerCache.value || {})) {
    const tt = (t || '').trim().toUpperCase()
    const nn = (n || '').trim()
    if (tt && nn && !m.has(tt)) {
      m.set(tt, nn)
    }
  }
  for (const s of [...watchlist.value, ...futuresWatchlist.value]) {
    const t = (s.ticker || '').trim().toUpperCase()
    const n = (s.name || '').trim()
    if (t && n && !m.has(t)) {
      m.set(t, n)
    }
  }
  return m
})

const visiblePortfolio = computed(() => {
  const items = portfolio.value || []
  if (showZeroPositions.value) return items
  return items.filter((p) => Number(p.quantity || 0) !== 0)
})

function rawTickerKey(ticker: string): string {
  const raw = (ticker || '').trim().toUpperCase()
  if (!raw) return ''
  if (raw.includes('@')) return raw.split('@')[0]
  if (raw.includes('_')) return raw.split('_')[0]
  return raw
}

function formatIssuerName(rawTicker: string): string {
  const key = rawTickerKey(rawTicker)
  if (!key) return ''
  const issuer = issuerByTicker.value.get(key)
  if (!issuer) return 'Эмитент: не определён'
  return `Эмитент: ${issuer}`
}

async function enrichIssuerCacheFromKnownTickers() {
  try {
    const tickerSet = new Set<string>()
    for (const p of portfolio.value) {
      const key = rawTickerKey(p.ticker)
      if (key) tickerSet.add(key)
    }
    for (const t of tradeJournal.value) {
      const key = rawTickerKey(t.ticker)
      if (key) tickerSet.add(key)
    }
    const unknown = Array.from(tickerSet).filter((t) => !issuerByTicker.value.has(t))
    if (unknown.length === 0) return

    const updates: Record<string, string> = {}
    for (const t of unknown.slice(0, 20)) {
      const [stocks, futures] = await Promise.all([
        GetStockInstruments(t, 10),
        GetFuturesInstruments(t, 10)
      ])
      const merged = [...(stocks || []), ...(futures || [])]
      const exact = merged.find((x) => (x.ticker || '').trim().toUpperCase() === t)
      if (exact?.name) {
        updates[t] = exact.name
      }
    }
    if (Object.keys(updates).length > 0) {
      issuerCache.value = { ...issuerCache.value, ...updates }
    }
  } catch (e) {
    console.warn('Cannot enrich issuer cache:', e)
  }
}

function splitRejectedStatus(status: string): { label: string; reason: string } {
  const s = (status || '').trim()
  const prefix = 'rejected:'
  if (s.toLowerCase().startsWith(prefix)) {
    const reason = s.slice(prefix.length).trim()
    return {
      label: 'rejected',
      reason: reason || '-'
    }
  }
  return {
    label: s || '-',
    reason: '-'
  }
}

function getTradeStatusLabel(status: string): string {
  return splitRejectedStatus(status).label
}

function getTradeStatusReason(status: string): string {
  return splitRejectedStatus(status).reason
}

function getTradeStatusKind(status: string): 'ok' | 'risk' | 'broker' | 'neutral' {
  const { label, reason } = splitRejectedStatus(status)
  const lowReason = reason.toLowerCase()
  const lowLabel = label.toLowerCase()

  if (lowLabel === 'placed' || lowLabel === 'executed') {
    return 'ok'
  }
  if (lowLabel !== 'rejected') {
    return 'neutral'
  }
  if (
    lowReason.includes('emergency stop') ||
    lowReason.includes('notional') ||
    lowReason.includes('daily') ||
    lowReason.includes('risk')
  ) {
    return 'risk'
  }
  return 'broker'
}

function getEventKind(level: string): 'ok' | 'risk' | 'broker' | 'neutral' {
  const low = (level || '').toLowerCase()
  if (low === 'info') return 'ok'
  if (low === 'warn' || low === 'warning') return 'risk'
  if (low === 'error') return 'broker'
  return 'neutral'
}

onMounted(() => {
  loadWatchlistCollapseState()
  loadAccounts()
  loadData()
  loadBlockedTickers()
  const interval = setInterval(loadData, 10000)
  onUnmounted(() => clearInterval(interval))
})

watch(autoInstrumentMode, (mode) => {
  if (mode !== 'fixed') {
    autoFixedTicker.value = ''
  }
})

watch(() => analyzeQuery.value, () => {
  updateAnalyzeCandidates()
})

watch(() => stocksQuery.value, () => {
  onWatchlistQueryChanged()
})

watch(() => futuresQuery.value, () => {
  onWatchlistQueryChanged()
})

watch(() => stocksWatchlistExpanded.value, (v) => {
  try {
    localStorage.setItem(stocksCollapseStorageKey, v ? '1' : '0')
  } catch {
    // ignore storage errors
  }
})

watch(() => futuresWatchlistExpanded.value, (v) => {
  try {
    localStorage.setItem(futuresCollapseStorageKey, v ? '1' : '0')
  } catch {
    // ignore storage errors
  }
})

watch(() => tradeJournalExpanded.value, (v) => {
  try {
    localStorage.setItem(tradeJournalCollapseStorageKey, v ? '1' : '0')
  } catch {
    // ignore storage errors
  }
})

watch(() => runtimeEventsSource.value, () => {
  loadRuntimeEvents()
})

watch(() => runtimeEventsLimit.value, () => {
  loadRuntimeEvents()
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  min-height: 100vh;
  color: #fff;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}

.header h1 {
  margin: 0;
  font-size: 28px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ff4444;
}

.dot.green {
  background: #44ff44;
}

.account-panel {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 15px;
  margin-bottom: 20px;
}

.account-selector {
  display: flex;
  gap: 10px;
  margin: 10px 0;
  flex-wrap: wrap;
}

.account-selector button {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  border-radius: 8px;
  color: white;
  cursor: pointer;
  transition: all 0.3s;
}

.account-selector button.active {
  background: #44ff44;
  color: #1a1a2e;
}

.account-selector button:hover {
  background: rgba(255, 255, 255, 0.3);
}

.current-account {
  font-size: 14px;
  opacity: 0.8;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.stat-card {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border-radius: 20px;
  padding: 20px;
  text-align: center;
}

.stat-card h3 {
  margin: 0 0 10px 0;
  font-size: 14px;
  opacity: 0.8;
}

.stat-card .value {
  font-size: 32px;
  font-weight: bold;
}

.value.positive {
  color: #44ff44;
}

.value.negative {
  color: #ff4444;
}

.balance-warning {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.35;
  color: #ffb347;
}

.stats-freshness {
  margin-top: -12px;
  margin-bottom: 18px;
  font-size: 12px;
  opacity: 0.9;
  display: flex;
  align-items: center;
  gap: 8px;
}

.stats-freshness .badge {
  border-radius: 999px;
  padding: 2px 8px;
  font-weight: bold;
  text-transform: lowercase;
}

.stats-freshness .badge.cached {
  background: rgba(250, 204, 21, 0.2);
  color: #fde68a;
}

.stats-freshness .badge.live {
  background: rgba(34, 197, 94, 0.22);
  color: #86efac;
}

.session-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
  margin-bottom: 18px;
}

.session-item {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 10px 12px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.session-label {
  opacity: 0.8;
}

.session-time {
  font-family: monospace;
  font-weight: bold;
}

.session-pill {
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  text-transform: lowercase;
}

.session-pill.open {
  background: rgba(34, 197, 94, 0.22);
  color: #86efac;
}

.session-pill.closed {
  background: rgba(239, 68, 68, 0.22);
  color: #fca5a5;
}

.session-note {
  opacity: 0.75;
  font-size: 12px;
}

.agent-panel {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 20px;
  padding: 20px;
  margin-bottom: 30px;
}

.mode-status {
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 13px;
  margin-bottom: 12px;
  font-weight: 600;
}

.mode-status.kind-agent {
  background: rgba(34, 197, 94, 0.2);
  color: #86efac;
}

.mode-status.kind-manual {
  background: rgba(250, 204, 21, 0.2);
  color: #fde68a;
}

.mode-status.kind-analysis {
  background: rgba(59, 130, 246, 0.22);
  color: #93c5fd;
}

.market-status {
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 13px;
  margin-bottom: 12px;
  font-weight: 600;
}

.market-status.kind-open {
  background: rgba(34, 197, 94, 0.2);
  color: #86efac;
}

.market-status.kind-partial {
  background: rgba(59, 130, 246, 0.22);
  color: #93c5fd;
}

.market-status.kind-clearing {
  background: rgba(245, 158, 11, 0.2);
  color: #fcd34d;
}

.market-status.kind-closed {
  background: rgba(239, 68, 68, 0.22);
  color: #fca5a5;
}

.market-hint {
  margin-bottom: 12px;
  font-size: 12px;
  opacity: 0.82;
  color: #dbeafe;
}

.silence-status {
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 13px;
  margin-bottom: 12px;
  font-weight: 600;
}

.silence-status.kind-ok {
  background: rgba(34, 197, 94, 0.18);
  color: #86efac;
}

.silence-status.kind-warn {
  background: rgba(245, 158, 11, 0.2);
  color: #fcd34d;
}

.silence-status.kind-risk {
  background: rgba(239, 68, 68, 0.2);
  color: #fca5a5;
}

.silence-status.kind-neutral {
  background: rgba(148, 163, 184, 0.18);
  color: #cbd5e1;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.agent-controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.panel-header h2 {
  margin: 0;
  font-size: 20px;
}

.auto-btn {
  background: #5a6cff;
  color: white;
  border: none;
  padding: 8px 14px;
  border-radius: 20px;
  cursor: pointer;
  font-weight: bold;
}

.auto-btn.active {
  background: #22c55e;
  color: #08230f;
}

.toggle-btn {
  background: #ff4444;
  color: white;
  border: none;
  padding: 8px 20px;
  border-radius: 20px;
  cursor: pointer;
  font-weight: bold;
}

.toggle-btn.active {
  background: #44ff44;
  color: #1a1a2e;
}

.analyze-form {
  display: flex;
  gap: 10px;
  margin-bottom: 15px;
}

.analyze-hidden-note {
  margin-bottom: 14px;
  font-size: 12px;
  opacity: 0.8;
}

.auto-instrument-config {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: flex-end;
  margin-bottom: 14px;
}

.auto-instrument-config label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  min-width: 240px;
}

.auto-instrument-config select {
  min-width: 240px;
  padding: 9px 12px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.auto-instrument-config input {
  min-width: 140px;
  padding: 9px 12px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.auto-instrument-config select option {
  color: #111;
}

.save-auto-config-btn {
  padding: 9px 14px;
  border: none;
  border-radius: 10px;
  font-weight: 600;
  cursor: pointer;
  background: #22c55e;
  color: #08230f;
}

.reset-soft-btn {
  background: #3b82f6;
  color: #eaf2ff;
}

.safety-presets {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  width: 100%;
  margin: 4px 0 8px;
}

.safety-presets-label {
  font-size: 13px;
  opacity: 0.92;
  width: 100%;
}

.preset-chip {
  font-size: 13px;
  padding: 8px 12px;
}

.analyze-form input {
  flex: 1;
  padding: 10px 15px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  font-size: 14px;
}

.analyze-form select {
  min-width: 240px;
  padding: 10px 15px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  font-size: 14px;
}

.analyze-form select option {
  color: #111;
}

.analyze-form input::placeholder {
  color: rgba(255, 255, 255, 0.5);
}

.analyze-form button {
  padding: 10px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  border-radius: 10px;
  color: white;
  cursor: pointer;
  font-weight: bold;
}

.analysis-result {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 12px;
  padding: 15px;
  margin-top: 15px;
}

.signal-action {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-weight: bold;
  margin-bottom: 10px;
}

.signal-action.buy {
  background: #44ff44;
  color: #1a1a2e;
}

.signal-action.sell {
  background: #ff4444;
  color: white;
}

.signal-action.hold {
  background: #ffaa44;
  color: #1a1a2e;
}

.signal-confidence {
  font-size: 12px;
  opacity: 0.7;
  margin-bottom: 8px;
}

.signal-reasoning {
  font-size: 14px;
}

.signal-error {
  color: #ff4444;
}

.market-tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.tab-btn {
  flex: 1;
  padding: 12px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 12px;
  color: white;
  font-size: 16px;
  cursor: pointer;
  transition: all 0.3s;
}

.tab-btn.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.tab-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.watchlist, .portfolio {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border-radius: 20px;
  padding: 20px;
  margin-bottom: 20px;
}

.watchlist h2, .portfolio h2 {
  margin: 0 0 15px 0;
  font-size: 18px;
}

.runtime-header {
  margin-bottom: 12px;
}

.runtime-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.runtime-controls label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  opacity: 0.9;
}

.runtime-controls select {
  min-width: 170px;
  padding: 7px 10px;
  border: none;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.runtime-controls select option {
  color: #111;
}

.runtime-refresh-btn {
  font-size: 16px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.collapse-btn {
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.35);
}

.watchlist-search {
  margin-bottom: 12px;
}

.watchlist-search input {
  width: 100%;
  padding: 10px 14px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  font-size: 14px;
}

.watchlist-search input::placeholder {
  color: rgba(255, 255, 255, 0.6);
}

.portfolio-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: 10px;
}

.checkbox-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  opacity: 0.9;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th, td {
  text-align: left;
  padding: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

th {
  opacity: 0.7;
  font-weight: normal;
  font-size: 12px;
}

.ticker {
  font-weight: bold;
  color: #667eea;
}

.ticker-meta {
  margin-top: 2px;
  font-size: 11px;
  color: rgba(226, 232, 240, 0.75);
  font-weight: 500;
}

.ticker-issuer {
  margin-top: 2px;
  font-size: 11px;
  color: rgba(148, 163, 184, 0.95);
  font-weight: 500;
}

.price {
  font-family: monospace;
}

.pnl.positive {
  color: #44ff44;
}

.pnl.negative {
  color: #ff4444;
}

button.buy, button.sell {
  padding: 4px 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  margin: 0 4px;
}

button.buy {
  background: #44ff44;
  color: #1a1a2e;
}

button.sell {
  background: #ff4444;
  color: white;
}

button.close-position {
  padding: 4px 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  margin: 0 4px;
  font-weight: 600;
}

button.close-long {
  background: #ff9500;
  color: white;
}

button.close-short {
  background: #9500ff;
  color: white;
}

button.sltp {
  margin-right: 8px;
  background: #f59e0b;
  color: #1a1a2e;
  font-weight: 600;
}

.loading {
  text-align: center;
  padding: 40px;
  opacity: 0.6;
}

.journal-source {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 6px;
  white-space: nowrap;
}

.journal-source.src-finam {
  background: rgba(59, 130, 246, 0.25);
  color: #93c5fd;
}

.journal-source.src-local {
  background: rgba(148, 163, 184, 0.2);
  color: #e2e8f0;
}

.trade-status {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  text-transform: lowercase;
}

.trade-status.kind-ok {
  background: rgba(34, 197, 94, 0.22);
  color: #86efac;
}

.trade-status.kind-risk {
  background: rgba(249, 115, 22, 0.2);
  color: #fdba74;
}

.trade-status.kind-broker {
  background: rgba(239, 68, 68, 0.22);
  color: #fca5a5;
}

.trade-status.kind-neutral {
  background: rgba(148, 163, 184, 0.2);
  color: #cbd5e1;
}

.trade-reason {
  max-width: 360px;
  color: rgba(255, 255, 255, 0.85);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>