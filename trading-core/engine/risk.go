package engine

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type RiskConfig struct {
	Enabled bool

	// MaxOrderNotional — максимальная стоимость одной заявки (в валюте котировки).
	MaxOrderNotional float64

	// MaxDailyNotional — суммарный лимит заявок на день (грубая оценка по notional).
	MaxDailyNotional float64
}

type OrderIntent struct {
	Symbol  string
	Side    Side
	Qty     float64
	Price   float64 // 0 => неизвестно (market), тогда caller должен подставить оценку
	Notional float64
	Time    time.Time
}

type RiskStatus struct {
	Enabled         bool    `json:"enabled"`
	EmergencyStop   bool    `json:"emergency_stop"`
	DailyNotional   float64 `json:"daily_notional"`
	MaxDailyNotional float64 `json:"max_daily_notional"`
	MaxOrderNotional float64 `json:"max_order_notional"`
	Day             string  `json:"day"`
}

type RiskManager struct {
	mu sync.Mutex

	cfg RiskConfig

	emergencyStop bool

	day          string
	dailyNotional float64
}

func NewRiskManager(cfg RiskConfig) *RiskManager {
	return &RiskManager{
		cfg: cfg,
	}
}

func (r *RiskManager) SetConfig(cfg RiskConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
}

func (r *RiskManager) SetEmergencyStop(stop bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emergencyStop = stop
}

func (r *RiskManager) Status(now time.Time) RiskStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rotateDayLocked(now)
	return RiskStatus{
		Enabled:          r.cfg.Enabled,
		EmergencyStop:    r.emergencyStop,
		DailyNotional:    r.dailyNotional,
		MaxDailyNotional: r.cfg.MaxDailyNotional,
		MaxOrderNotional: r.cfg.MaxOrderNotional,
		Day:              r.day,
	}
}

func (r *RiskManager) Allow(intent OrderIntent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.cfg.Enabled {
		return nil
	}

	if r.emergencyStop {
		return errors.New("emergency stop is active")
	}

	r.rotateDayLocked(intent.Time)

	if intent.Notional <= 0 {
		return fmt.Errorf("invalid notional: %.6f", intent.Notional)
	}

	if r.cfg.MaxOrderNotional > 0 && intent.Notional > r.cfg.MaxOrderNotional {
		return fmt.Errorf("order notional %.2f exceeds max %.2f", intent.Notional, r.cfg.MaxOrderNotional)
	}

	if r.cfg.MaxDailyNotional > 0 && (r.dailyNotional+intent.Notional) > r.cfg.MaxDailyNotional {
		return fmt.Errorf("daily notional %.2f would exceed max %.2f", r.dailyNotional+intent.Notional, r.cfg.MaxDailyNotional)
	}

	// резервируем лимит в момент отправки заявки
	r.dailyNotional += intent.Notional
	return nil
}

func (r *RiskManager) rotateDayLocked(now time.Time) {
	day := now.Format("2006-01-02")
	if r.day == "" {
		r.day = day
		return
	}
	if r.day != day {
		r.day = day
		r.dailyNotional = 0
	}
}

