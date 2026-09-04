package moex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type Price struct {
	Ticker string
	Value  float64
	Time   time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://iss.moex.com/iss",
	}
}

func (c *Client) GetCurrentPrice(ticker string) (float64, error) {
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

	// Парсим marketdata
	if marketdata, ok := result["marketdata"].(map[string]interface{}); ok {
		if data, ok := marketdata["data"].([]interface{}); ok && len(data) > 0 {
			if row, ok := data[0].([]interface{}); ok && len(row) > 0 {
				// Ищем колонку с ценой
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

func (c *Client) GetTickers() []string {
	return []string{
		"SBER", "GAZP", "LKOH", "YNDX", "TATN",
		"ROSN", "VTBR", "NVTK", "PLZL", "MGNT",
		"CHMF", "NLMK", "AFKS", "MTSS", "RUAL",
	}
}
