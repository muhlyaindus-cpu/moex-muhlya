package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MOEXClient struct {
	httpClient *http.Client
	baseURL    string
}

type Security struct {
	Ticker string  `json:"ticker"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
}

func NewMOEXClient() *MOEXClient {
	return &MOEXClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://iss.moex.com/iss",
	}
}

func (c *MOEXClient) GetCurrentPrice(ticker string) (float64, error) {
	url := fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s.json",
		c.baseURL, ticker)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if marketdata, ok := result["marketdata"].(map[string]interface{}); ok {
		if data, ok := marketdata["data"].([]interface{}); ok && len(data) > 0 {
			if row, ok := data[0].([]interface{}); ok && len(row) > 0 {
				if columns, ok := marketdata["columns"].([]interface{}); ok {
					for i, col := range columns {
						if col == "LAST" && i < len(row) {
							if price, ok := row[i].(float64); ok {
								return price, nil
							}
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("не удалось получить цену для %s", ticker)
}

func (c *MOEXClient) GetWatchlist() []Security {
	tickers := []string{"SBER", "GAZP", "LKOH", "YNDX", "TATN", "ROSN", "VTBR"}
	securities := make([]Security, 0)

	for _, ticker := range tickers {
		price, err := c.GetCurrentPrice(ticker)
		if err == nil {
			securities = append(securities, Security{
				Ticker: ticker,
				Name:   getTickerName(ticker),
				Price:  price,
			})
		}
	}

	return securities
}

func getTickerName(ticker string) string {
	names := map[string]string{
		"SBER": "Сбербанк",
		"GAZP": "Газпром",
		"LKOH": "Лукойл",
		"YNDX": "Яндекс",
		"TATN": "Татнефть",
		"ROSN": "Роснефть",
		"VTBR": "ВТБ",
	}
	return names[ticker]
}
