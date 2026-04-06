package scraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// StockData holds cleaned OHLCV history and metadata from Yahoo Finance.
type StockData struct {
	Symbol       string
	Currency     string
	CurrentPrice float64
	Week52High   float64
	Week52Low    float64
	Timestamps   []int64
	Opens        []float64
	Highs        []float64
	Lows         []float64
	Closes       []float64
	Volumes      []float64
}

type yahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				FiftyTwoWeekHigh   float64 `json:"fiftyTwoWeekHigh"`
				FiftyTwoWeekLow    float64 `json:"fiftyTwoWeekLow"`
				Currency           string  `json:"currency"`
				Symbol             string  `json:"symbol"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

var client = &http.Client{Timeout: 15 * time.Second}

// FetchYahoo retrieves 1 year of daily OHLCV data for the given ticker.
func FetchYahoo(ticker string) (*StockData, error) {
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1y&includePrePost=false",
		ticker,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("yahoo returned status %d — check ticker symbol", resp.StatusCode)
	}

	var yr yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&yr); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}

	if yr.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", yr.Chart.Error)
	}
	if len(yr.Chart.Result) == 0 {
		return nil, errors.New("no data returned — check ticker symbol")
	}

	r := yr.Chart.Result[0]
	if len(r.Indicators.Quote) == 0 {
		return nil, errors.New("no quote data in response")
	}
	q := r.Indicators.Quote[0]

	sd := &StockData{
		Symbol:       r.Meta.Symbol,
		Currency:     r.Meta.Currency,
		CurrentPrice: r.Meta.RegularMarketPrice,
		Week52High:   r.Meta.FiftyTwoWeekHigh,
		Week52Low:    r.Meta.FiftyTwoWeekLow,
	}

	// Zip slices, skip bars with any nil or zero value
	for i := range r.Timestamp {
		if i >= len(q.High) || i >= len(q.Low) || i >= len(q.Close) {
			break
		}
		if q.High[i] == nil || q.Low[i] == nil || q.Close[i] == nil {
			continue
		}
		h, l, c := *q.High[i], *q.Low[i], *q.Close[i]
		if h == 0 || l == 0 || c == 0 {
			continue
		}
		o := 0.0
		if i < len(q.Open) && q.Open[i] != nil {
			o = *q.Open[i]
		}
		v := 0.0
		if i < len(q.Volume) && q.Volume[i] != nil {
			v = *q.Volume[i]
		}
		sd.Timestamps = append(sd.Timestamps, r.Timestamp[i])
		sd.Opens = append(sd.Opens, o)
		sd.Highs = append(sd.Highs, h)
		sd.Lows = append(sd.Lows, l)
		sd.Closes = append(sd.Closes, c)
		sd.Volumes = append(sd.Volumes, v)
	}

	if len(sd.Closes) < 15 {
		return nil, errors.New("insufficient history — need at least 15 trading days")
	}
	return sd, nil
}
