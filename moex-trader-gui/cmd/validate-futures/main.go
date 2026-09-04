// Проверка фьючерсных тикеров (MOEX SECID / как в lchi_futures_allowlist) через тот же путь, что Muhlya:
// gRPC GetAsset + кандидаты из finamSymbolCandidatesForAPI.
//
// Запуск: из каталога с .env (например /opt/trading/app/moex-trader-gui):
//   go run ./cmd/validate-futures SiM6 BRM6 GZM6
//   go run ./cmd/validate-futures -f tickers.txt
//   go build -o validate-futures ./cmd/validate-futures
//
// Требуется FINAM_SECRET_REAL. Счёт — как в брокере (токен / FINAM_ACCOUNT_ID).

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"moex-trader-gui/services"

	"github.com/joho/godotenv"
)

func main() {
	var file = flag.String("f", "", "файл: по одному тикеру в строке (# — комментарий)")
	var allowlistFragment = flag.Bool("allowlist-fragment", false, "печатать только строки для lchi (по одному тикеру, только ok)")
	flag.Parse()
	args := filterEmpty(flag.Args())

	if *file != "" {
		lines, err := readTickersFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "file: %v\n", err)
			os.Exit(1)
		}
		args = append(args, lines...)
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: validate-futures [-f file] [-allowlist-fragment] ticker [ticker...]\n")
		os.Exit(1)
	}

	_ = godotenv.Load()
	if strings.TrimSpace(os.Getenv("AGENT_FINAM_GETASSET_PREFLIGHT")) == "0" {
		fmt.Fprintf(os.Stderr, "warning: AGENT_FINAM_GETASSET_PREFLIGHT=0 — брокер пропустит preflight; результат неинформативен\n")
	}
	secret := os.Getenv("FINAM_SECRET_REAL")
	if secret == "" {
		fmt.Fprintf(os.Stderr, "FINAM_SECRET_REAL is not set\n")
		os.Exit(1)
	}
	b, err := services.NewFinamBroker(secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "broker: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	ctx := context.Background()
	if *allowlistFragment {
		for _, t := range args {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			ok, _ := b.ValidateFuturesSymbolForAccount(ctx, t)
			if ok {
				fmt.Println(t)
			}
		}
		return
	}

	fmt.Printf("%-20s  %-4s  %s\n", "ticker", "ok", "detail")
	for _, t := range args {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		ok, detail := b.ValidateFuturesSymbolForAccount(ctx, t)
		o := "no"
		if ok {
			o = "yes"
		}
		if detail == "" {
			detail = "—"
		}
		fmt.Printf("%-20s  %-4s  %s\n", t, o, detail)
	}
}

func filterEmpty(s []string) []string {
	var out []string
	for _, x := range s {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func readTickersFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}
