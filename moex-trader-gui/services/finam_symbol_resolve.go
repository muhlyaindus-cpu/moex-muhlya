package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	assetspb "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"google.golang.org/grpc/metadata"
)

// brokerSymbolOverrideKey — ключ для карты переопределений: верхний регистр, без @MIC и без суффикса _TQBR/_SPBFUT.
func brokerSymbolOverrideKey(ticker string) string {
	t := strings.TrimSpace(ticker)
	if t == "" {
		return ""
	}
	if i := strings.Index(t, "@"); i > 0 {
		t = t[:i]
	}
	if i := strings.Index(t, "_"); i > 0 {
		t = t[:i]
	}
	return strings.ToUpper(strings.TrimSpace(t))
}

// normalizeFuturesMatchKey — сопоставление MOEX SECID и тикера Finam (без дефисов/точек, верхний регистр).
func normalizeFuturesMatchKey(ticker string) string {
	t := brokerSymbolOverrideKey(ticker)
	t = strings.ReplaceAll(t, "-", "")
	t = strings.ReplaceAll(t, ".", "")
	return strings.ToUpper(strings.TrimSpace(t))
}

func envFuturesIndexTTL() time.Duration {
	def := 6 * time.Hour
	raw := strings.TrimSpace(os.Getenv("AGENT_FINAM_FUTURES_INDEX_TTL_HOURS"))
	if raw == "" {
		return def
	}
	h, err := strconv.ParseFloat(raw, 64)
	if err != nil || h <= 0 {
		return def
	}
	return time.Duration(h * float64(time.Hour))
}

// envGetAssetNotFoundCacheSec — TTL негативного кэша GetAsset при NotFound/«нет инструмента» (сек).
// По умолчанию 120. 0 = не кэшировать неудачу для таких ошибок (каждый раз новый gRPC).
func envGetAssetNotFoundCacheSec() time.Duration {
	def := 120 * time.Second
	raw := strings.TrimSpace(os.Getenv("AGENT_FINAM_GETASSET_NOTFOUND_CACHE_SEC"))
	if raw == "" {
		return def
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec < 0 {
		return def
	}
	return time.Duration(sec) * time.Second
}

func isFinamGetAssetNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "security id") ||
		strings.Contains(msg, "code = notfound")
}

// finamAssetInvalidCacheTTL — длинный кэш для сетевых/прочих ошибок; короткий при «инструмента нет».
func finamAssetInvalidCacheTTL(err error, emptyAssetResp bool) time.Duration {
	short := envGetAssetNotFoundCacheSec()
	if emptyAssetResp {
		if short == 0 {
			return 0
		}
		return short
	}
	if isFinamGetAssetNotFoundLike(err) {
		if short == 0 {
			return 0
		}
		return short
	}
	return 30 * time.Minute
}

// ensureFinamFuturesSymbolIndex — загрузка списка инструментов Finam (AllAssets, MIC=FUT) для сопоставления с MOEX.
// Отключить: AGENT_FINAM_FUTURES_SYMBOL_INDEX=0
func (b *FinamBroker) ensureFinamFuturesSymbolIndex(ctx context.Context) {
	if strings.TrimSpace(os.Getenv("AGENT_FINAM_FUTURES_SYMBOL_INDEX")) == "0" {
		return
	}
	ttl := envFuturesIndexTTL()
	b.futuresIndexMu.Lock()
	if len(b.futuresSymbolByKey) > 0 && time.Since(b.futuresIndexAt) < ttl {
		b.futuresIndexMu.Unlock()
		return
	}
	b.futuresIndexMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	conn, err := b.ensureFinamGRPC(ctx)
	if err != nil {
		log.Printf("⚠️ Finam futures index: grpc: %v", err)
		return
	}
	jwt, err := b.getJWT()
	if err != nil {
		log.Printf("⚠️ Finam futures index: jwt: %v", err)
		return
	}
	auth := finamAuthorizationHeader(jwt)
	if auth == "" {
		return
	}
	mdCtx := metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
	cli := assetspb.NewAssetsServiceClient(conn)

	newMap := make(map[string]string, 16384)
	var cursor int64
	pages := 0
	for {
		pages++
		if pages > 500 {
			log.Printf("⚠️ Finam AllAssets: слишком много страниц, обрыв")
			break
		}
		resp, err := cli.AllAssets(mdCtx, &assetspb.AllAssetsRequest{
			Cursor:     cursor,
			OnlyActive: true,
		})
		if err != nil {
			log.Printf("⚠️ Finam AllAssets (cursor=%d): %v", cursor, err)
			return
		}
		if resp == nil {
			break
		}
		for _, a := range resp.GetAssets() {
			if a == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(a.GetMic()), "FUT") {
				continue
			}
			sym := strings.TrimSpace(a.GetSymbol())
			if sym == "" {
				continue
			}
			tk := strings.TrimSpace(a.GetTicker())
			name := strings.TrimSpace(a.GetName())
			for _, raw := range []string{tk, sym, strings.TrimSuffix(sym, "@FUT"), name} {
				k := normalizeFuturesMatchKey(raw)
				if k == "" {
					continue
				}
				if _, ok := newMap[k]; !ok {
					newMap[k] = sym
				}
			}
		}
		cursor = resp.GetNextCursor()
		if cursor == 0 {
			break
		}
	}
	if len(newMap) == 0 {
		log.Printf("⚠️ Finam futures index: пустой ответ AllAssets, прежний кэш не трогаем")
		return
	}
	b.futuresIndexMu.Lock()
	b.futuresSymbolByKey = newMap
	b.futuresIndexAt = time.Now()
	b.futuresIndexMu.Unlock()
	// После обновления справочника FUT старые негативные GetAsset по прежним символам теряют смысл.
	b.finamAssetMu.Lock()
	b.assetInvalidUntil = make(map[string]time.Time)
	b.finamAssetMu.Unlock()
	log.Printf("📎 Finam futures symbol index: %d ключей (MIC=FUT, справочник ai.finam.ru/futures)", len(newMap))
}

// resolveFinamSymbolOverridesPath — явный путь из FINAM_SYMBOL_OVERRIDES_PATH или авто: рядом с cwd / с exe.
// Приоритет файлов: finam_symbol_overrides.json (личный), затем finam_symbol_overrides_screener.json (дамп скринера).
func resolveFinamSymbolOverridesPath() string {
	if p := strings.TrimSpace(os.Getenv("FINAM_SYMBOL_OVERRIDES_PATH")); p != "" {
		return p
	}
	var candidates []string
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "finam_symbol_overrides.json"),
			filepath.Join(wd, "finam_symbol_overrides_screener.json"),
		)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "finam_symbol_overrides.json"),
			filepath.Join(dir, "finam_symbol_overrides_screener.json"),
		)
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// LoadFinamSymbolOverridesFromEnv загружает JSON из FINAM_SYMBOL_OVERRIDES_PATH или из авто-пути (см. resolveFinamSymbolOverridesPath).
// Формат: {"AFU6":"AFU6@FUT","Si6":"Si-6.25@FUT"} — ключ как в MOEX ISS, значение — как в Finam Trade API.
func LoadFinamSymbolOverridesFromEnv() map[string]string {
	path := resolveFinamSymbolOverridesPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("⚠️ FINAM_SYMBOL_OVERRIDES_PATH: не удалось прочитать %s: %v", path, err)
		return nil
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("⚠️ FINAM_SYMBOL_OVERRIDES_PATH: неверный JSON %s: %v", path, err)
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		k = strings.TrimSpace(k)
		if k == "" || strings.HasPrefix(strings.ToLower(k), "_") {
			continue
		}
		kk := brokerSymbolOverrideKey(k)
		vv := strings.TrimSpace(v)
		if kk == "" || vv == "" {
			continue
		}
		out[kk] = vv
	}
	if len(out) == 0 {
		return nil
	}
	log.Printf("📎 Загружено переопределений символов Finam: %d (из %s)", len(out), path)
	return out
}

func finamSymbolAliasCandidates(symbol string) []string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(symbol)
	if strings.HasSuffix(strings.ToUpper(symbol), "@FUT") {
		add(strings.TrimSpace(strings.TrimSuffix(symbol, "@FUT")))
	}
	if strings.HasSuffix(strings.ToUpper(symbol), "@MISX") {
		add(strings.TrimSpace(strings.TrimSuffix(symbol, "@MISX")))
	}
	return out
}

// finamSymbolCandidatesForAPI — символы для preflight GetAsset (основной + запасной MOEX-стиль TICKER@FUT).
func (b *FinamBroker) finamSymbolCandidatesForAPI(ticker string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range finamSymbolAliasCandidates(b.OrderSymbolForAPI(ticker)) {
		add(s)
	}
	if isLikelyFutureTicker(ticker) {
		add(b.formatSymbolForREST(ticker))
	}
	return out
}

// rememberFinamFutureSymbolResolution — запоминает символ, прошедший GetAsset (ключ — нормализованный MOEX SECID).
func (b *FinamBroker) rememberFinamFutureSymbolResolution(ticker, resolvedSymbol string) {
	if strings.TrimSpace(os.Getenv("AGENT_FINAM_FUTURES_SYMBOL_LEARN")) == "0" {
		return
	}
	nk := normalizeFuturesMatchKey(ticker)
	rs := strings.TrimSpace(resolvedSymbol)
	if nk == "" || rs == "" {
		return
	}
	b.futuresLearnedMu.Lock()
	if b.futuresSymbolLearned == nil {
		b.futuresSymbolLearned = make(map[string]string)
	}
	b.futuresSymbolLearned[nk] = rs
	b.futuresLearnedMu.Unlock()
}

// OrderSymbolForAPI возвращает полный символ для REST/WS (с учётом переопределений).
func (b *FinamBroker) OrderSymbolForAPI(ticker string) string {
	key := brokerSymbolOverrideKey(ticker)
	if key != "" && b.symbolOverrides != nil {
		if v, ok := b.symbolOverrides[key]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	if isLikelyFutureTicker(ticker) {
		nk := normalizeFuturesMatchKey(ticker)
		b.futuresLearnedMu.RLock()
		if sym, ok := b.futuresSymbolLearned[nk]; ok && strings.TrimSpace(sym) != "" {
			b.futuresLearnedMu.RUnlock()
			return strings.TrimSpace(sym)
		}
		b.futuresLearnedMu.RUnlock()
		b.ensureFinamFuturesSymbolIndex(context.Background())
		b.futuresIndexMu.RLock()
		if sym, ok := b.futuresSymbolByKey[nk]; ok && strings.TrimSpace(sym) != "" {
			b.futuresIndexMu.RUnlock()
			return strings.TrimSpace(sym)
		}
		b.futuresIndexMu.RUnlock()
	}
	return b.formatSymbolForREST(ticker)
}

// ValidateFuturesSymbolForAccount — gRPC GetAsset: есть ли инструмент у счёта в Finam.
// Отключается: AGENT_FINAM_GETASSET_PREFLIGHT=0
func (b *FinamBroker) ValidateFuturesSymbolForAccount(ctx context.Context, ticker string) (ok bool, detail string) {
	if strings.TrimSpace(os.Getenv("AGENT_FINAM_GETASSET_PREFLIGHT")) == "0" {
		return true, ""
	}
	cands := b.finamSymbolCandidatesForAPI(ticker)
	if len(cands) == 0 {
		return false, "no finam symbol candidates"
	}
	conn, err := b.ensureFinamGRPC(ctx)
	if err != nil {
		return false, err.Error()
	}
	jwt, err := b.getJWT()
	if err != nil {
		return false, err.Error()
	}
	auth := finamAuthorizationHeader(jwt)
	if auth == "" {
		return false, "empty jwt"
	}
	mdCtx := metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
	aid := normalizeFinamAccountIDForREST(b.GetCurrentAccount())
	if aid == "" {
		return false, "no current account"
	}
	cli := assetspb.NewAssetsServiceClient(conn)

	var lastDetail string
	for i, sym := range cands {
		now := time.Now()
		b.finamAssetMu.Lock()
		if t, ok := b.assetValidUntil[sym]; ok && now.Before(t) {
			b.finamAssetMu.Unlock()
			return true, ""
		}
		if t, ok := b.assetInvalidUntil[sym]; ok && now.Before(t) {
			b.finamAssetMu.Unlock()
			lastDetail = "cached: finam getasset miss"
			continue
		}
		b.finamAssetMu.Unlock()
		resp, err := cli.GetAsset(mdCtx, &assetspb.GetAssetRequest{
			Symbol:    sym,
			AccountId: aid,
		})
		if err != nil {
			if ttl := finamAssetInvalidCacheTTL(err, false); ttl > 0 {
				b.finamAssetMu.Lock()
				b.assetInvalidUntil[sym] = time.Now().Add(ttl)
				b.finamAssetMu.Unlock()
			}
			lastDetail = err.Error()
			continue
		}
		if resp == nil || (strings.TrimSpace(resp.GetId()) == "" && strings.TrimSpace(resp.GetTicker()) == "") {
			if ttl := finamAssetInvalidCacheTTL(nil, true); ttl > 0 {
				b.finamAssetMu.Lock()
				b.assetInvalidUntil[sym] = time.Now().Add(ttl)
				b.finamAssetMu.Unlock()
			}
			lastDetail = "empty asset id/ticker in response"
			continue
		}
		b.finamAssetMu.Lock()
		b.assetValidUntil[sym] = time.Now().Add(24 * time.Hour)
		b.finamAssetMu.Unlock()
		if i > 0 {
			b.rememberFinamFutureSymbolResolution(ticker, sym)
			log.Printf("📎 Finam GetAsset: для %s подошёл альтернативный символ %s (первая попытка %s)", ticker, sym, cands[0])
		}
		return true, ""
	}
	if lastDetail == "" {
		lastDetail = "no finam symbol candidates"
	}
	return false, lastDetail
}
