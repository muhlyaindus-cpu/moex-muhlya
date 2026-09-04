package services

import "math"

func SMA(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// RSI14 вычисляет RSI по классической формуле Wilder для последнего значения.
// Требует минимум period+1 значений.
func RSI(values []float64, period int) float64 {
	if period <= 0 || len(values) < period+1 {
		return 0
	}

	gain := 0.0
	loss := 0.0
	for i := len(values) - period; i < len(values); i++ {
		delta := values[i] - values[i-1]
		if delta > 0 {
			gain += delta
		} else {
			loss += -delta
		}
	}

	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)
	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))
	if math.IsNaN(rsi) || math.IsInf(rsi, 0) {
		return 0
	}
	return rsi
}

func EMA(values []float64, period int) float64 {
	if period <= 0 || len(values) == 0 {
		return 0
	}
	if len(values) < period {
		return SMA(values)
	}
	mult := 2.0 / float64(period+1)
	ema := SMA(values[:period])
	for i := period; i < len(values); i++ {
		ema = (values[i]-ema)*mult + ema
	}
	return ema
}

func MACD(values []float64, fast, slow, signal int) (float64, float64, float64) {
	if len(values) < slow+signal || fast <= 0 || slow <= 0 || signal <= 0 || fast >= slow {
		return 0, 0, 0
	}
	macdSeries := make([]float64, 0, len(values)-slow+1)
	for i := slow; i <= len(values); i++ {
		w := values[:i]
		macdSeries = append(macdSeries, EMA(w, fast)-EMA(w, slow))
	}
	if len(macdSeries) == 0 {
		return 0, 0, 0
	}
	macd := macdSeries[len(macdSeries)-1]
	sig := EMA(macdSeries, signal)
	return macd, sig, macd - sig
}

// VolatilityPctByCloses возвращает прокси-оценку волатильности в долях цены.
// Используем средний абсолютный ход между соседними закрытиями.
func VolatilityPctByCloses(values []float64, period int) float64 {
	if period <= 1 || len(values) < period+1 {
		return 0
	}
	start := len(values) - period
	ref := values[len(values)-1]
	if ref <= 0 {
		return 0
	}
	sumAbsMove := 0.0
	steps := 0
	for i := start; i < len(values); i++ {
		move := math.Abs(values[i] - values[i-1])
		sumAbsMove += move
		steps++
	}
	if steps == 0 {
		return 0
	}
	return (sumAbsMove / float64(steps)) / ref
}
