export namespace main {
	
	export class AutoDecisionStatus {
	    at: string;
	    universe_size: number;
	    passed_filters: number;
	    blocked_by_cooldown: number;
	    blocked_by_qualification: number;
	    blocked_by_rotation: number;
	    blocked_by_cash: number;
	    rejected_by_liquidity: number;
	    rejected_by_price: number;
	    best_buy_ticker: string;
	    best_buy_conf: string;
	    best_sell_ticker: string;
	    best_sell_conf: string;
	    action_taken: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoDecisionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.universe_size = source["universe_size"];
	        this.passed_filters = source["passed_filters"];
	        this.blocked_by_cooldown = source["blocked_by_cooldown"];
	        this.blocked_by_qualification = source["blocked_by_qualification"];
	        this.blocked_by_rotation = source["blocked_by_rotation"];
	        this.blocked_by_cash = source["blocked_by_cash"];
	        this.rejected_by_liquidity = source["rejected_by_liquidity"];
	        this.rejected_by_price = source["rejected_by_price"];
	        this.best_buy_ticker = source["best_buy_ticker"];
	        this.best_buy_conf = source["best_buy_conf"];
	        this.best_sell_ticker = source["best_sell_ticker"];
	        this.best_sell_conf = source["best_sell_conf"];
	        this.action_taken = source["action_taken"];
	        this.note = source["note"];
	    }
	}
	export class BlockedTicker {
	    ticker: string;
	    reason: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new BlockedTicker(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticker = source["ticker"];
	        this.reason = source["reason"];
	        this.type = source["type"];
	    }
	}
	export class RuntimeEvent {
	    at: string;
	    level: string;
	    step: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.level = source["level"];
	        this.step = source["step"];
	        this.details = source["details"];
	    }
	}
	export class SafetyConfig {
	    bad_ticker_cooldown_min: number;
	    auto_budget_percent: number;
	    auto_tp_pct: number;
	    auto_sl_pct: number;
	    auto_profit_lock_trigger_pct: number;
	    auto_profit_lock_trail_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new SafetyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bad_ticker_cooldown_min = source["bad_ticker_cooldown_min"];
	        this.auto_budget_percent = source["auto_budget_percent"];
	        this.auto_tp_pct = source["auto_tp_pct"];
	        this.auto_sl_pct = source["auto_sl_pct"];
	        this.auto_profit_lock_trigger_pct = source["auto_profit_lock_trigger_pct"];
	        this.auto_profit_lock_trail_pct = source["auto_profit_lock_trail_pct"];
	    }
	}
	export class TickerRejectStats {
	    ticker: string;
	    total_24h: number;
	    rejected_24h: number;
	    reject_rate_24h: number;
	    total_7d: number;
	    rejected_7d: number;
	    reject_rate_7d: number;
	
	    static createFrom(source: any = {}) {
	        return new TickerRejectStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticker = source["ticker"];
	        this.total_24h = source["total_24h"];
	        this.rejected_24h = source["rejected_24h"];
	        this.reject_rate_24h = source["reject_rate_24h"];
	        this.total_7d = source["total_7d"];
	        this.rejected_7d = source["rejected_7d"];
	        this.reject_rate_7d = source["reject_rate_7d"];
	    }
	}
	export class TradeJournalEntry {
	    id: number;
	    ticker: string;
	    side: string;
	    quantity: number;
	    price: number;
	    status: string;
	    executed_at: string;
	    source?: string;
	    external_id?: string;
	    realized_pl?: number;
	
	    static createFrom(source: any = {}) {
	        return new TradeJournalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticker = source["ticker"];
	        this.side = source["side"];
	        this.quantity = source["quantity"];
	        this.price = source["price"];
	        this.status = source["status"];
	        this.executed_at = source["executed_at"];
	        this.source = source["source"];
	        this.external_id = source["external_id"];
	        this.realized_pl = source["realized_pl"];
	    }
	}

}

export namespace services {
	
	export class Account {
	    ID: string;
	    Name: string;
	    Type: string;
	    TradingCode: string;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.TradingCode = source["TradingCode"];
	    }
	}
	export class Position {
	    ticker: string;
	    quantity: number;
	    avg_price: number;
	    current_price: number;
	    pnl: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticker = source["ticker"];
	        this.quantity = source["quantity"];
	        this.avg_price = source["avg_price"];
	        this.current_price = source["current_price"];
	        this.pnl = source["pnl"];
	    }
	}
	export class Security {
	    ticker: string;
	    name: string;
	    price: number;
	    num_trades?: number;
	    turnover_rub?: number;
	    // Go type: time
	    expiry_date?: any;
	
	    static createFrom(source: any = {}) {
	        return new Security(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticker = source["ticker"];
	        this.name = source["name"];
	        this.price = source["price"];
	        this.num_trades = source["num_trades"];
	        this.turnover_rub = source["turnover_rub"];
	        this.expiry_date = this.convertValues(source["expiry_date"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

