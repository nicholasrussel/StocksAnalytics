// Package fmp provides a client for the Financial Modeling Prep stable API.
// Base: https://financialmodelingprep.com/stable/{endpoint}?symbol=TICKER&apikey=KEY
//
// To call any endpoint not yet wrapped by a convenience method:
//   var result MyType
//   err := client.Fetch(ctx, "some-endpoint", fmp.P{"symbol": "AAPL"}, &result)
package fmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const stableBase = "https://financialmodelingprep.com/stable"
const v3Base = "https://financialmodelingprep.com/api/v3"

// P is a shorthand for endpoint query parameters.
type P = map[string]string

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Fetch calls any FMP stable endpoint.
// endpoint: e.g. "income-statement", "balance-sheet-statement", "ratios"
// params:   e.g. fmp.P{"symbol": "AAPL", "limit": "4", "period": "annual"}
func (c *Client) Fetch(ctx context.Context, endpoint string, params P, out any) error {
	u := stableBase + "/" + endpoint + "?apikey=" + url.QueryEscape(c.apiKey)
	for k, v := range params {
		u += "&" + k + "=" + url.QueryEscape(v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fmp: %s returned status %d", endpoint, resp.StatusCode)
	}

	// Peek at raw body to catch FMP error responses (returned with 200 but contain error fields).
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("fmp: %s invalid JSON: %w", endpoint, err)
	}
	// FMP signals "not in plan" or bad params via {"Error Message":"..."} or {"message":"..."}
	if len(raw) > 0 && raw[0] == '{' {
		var errResp struct {
			ErrorMessage string `json:"Error Message"`
			Message      string `json:"message"`
		}
		if json.Unmarshal(raw, &errResp) == nil {
			if errResp.ErrorMessage != "" {
				return fmt.Errorf("fmp: %s: %s", endpoint, errResp.ErrorMessage)
			}
			if errResp.Message != "" {
				return fmt.Errorf("fmp: %s: %s", endpoint, errResp.Message)
			}
		}
	}
	return json.Unmarshal(raw, out)
}

// ── Convenience methods ────────────────────────────────────────────────────────

// IncomeStatement is one period of income statement data.
type IncomeStatement struct {
	Date                 string  `json:"date"`
	Period               string  `json:"period"`
	Revenue              float64 `json:"revenue"`
	GrossProfit          float64 `json:"grossProfit"`
	OperatingIncome      float64 `json:"operatingIncome"`
	EBITDA               float64 `json:"ebitda"`
	NetIncome            float64 `json:"netIncome"`
	EPS                  float64 `json:"eps"`
	EPSDiluted           float64 `json:"epsdiluted"`
	GrossProfitRatio     float64 `json:"grossProfitRatio"`
	OperatingIncomeRatio float64 `json:"operatingIncomeRatio"`
	NetIncomeRatio       float64 `json:"netIncomeRatio"`
	InterestExpense      float64 `json:"interestExpense"`
	IncomeTaxExpense     float64 `json:"incomeTaxExpense"`
	WeightedAvgSharesOut float64 `json:"weightedAverageShsOut"`
}

// BalanceSheet is one period of balance sheet data.
type BalanceSheet struct {
	Date                    string  `json:"date"`
	Period                  string  `json:"period"`
	TotalAssets             float64 `json:"totalAssets"`
	TotalCurrentAssets      float64 `json:"totalCurrentAssets"`
	CashAndEquivalents      float64 `json:"cashAndCashEquivalents"`
	Inventory               float64 `json:"inventory"`
	TotalLiabilities        float64 `json:"totalLiabilities"`
	TotalCurrentLiabilities float64 `json:"totalCurrentLiabilities"`
	LongTermDebt            float64 `json:"longTermDebt"`
	ShortTermDebt           float64 `json:"shortTermDebt"`
	TotalDebt               float64 `json:"totalDebt"`
	TotalStockholdersEquity float64 `json:"totalStockholdersEquity"`
	RetainedEarnings        float64 `json:"retainedEarnings"`
}

// CashFlowStatement is one period of cash flow data.
type CashFlowStatement struct {
	Date                     string  `json:"date"`
	Period                   string  `json:"period"`
	OperatingCashFlow        float64 `json:"operatingCashFlow"`
	CapitalExpenditure       float64 `json:"capitalExpenditure"`
	FreeCashFlow             float64 `json:"freeCashFlow"`
	DividendsPaid            float64 `json:"dividendsPaid"`
	StockBasedCompensation   float64 `json:"stockBasedCompensation"`
	DepreciationAmortization float64 `json:"depreciationAndAmortization"`
}

// Quote holds real-time price and market cap.
type Quote struct {
	Symbol             string  `json:"symbol"`
	Price              float64 `json:"price"`
	ChangesPercentage  float64 `json:"changesPercentage"`
	MarketCap          float64 `json:"marketCap"`
	PE                 float64 `json:"pe"`
	EPS                float64 `json:"eps"`
	Beta               float64 `json:"beta"`
	YearHigh           float64 `json:"yearHigh"`
	YearLow            float64 `json:"yearLow"`
	SharesOutstanding  float64 `json:"sharesOutstanding"`
}

func (c *Client) GetIncomeStatements(ctx context.Context, ticker string, limit int) ([]IncomeStatement, error) {
	var result []IncomeStatement
	err := c.Fetch(ctx, "income-statement", P{
		"symbol": ticker,
		"limit":  fmt.Sprintf("%d", limit),
		"period": "annual",
	}, &result)
	return result, err
}

func (c *Client) GetBalanceSheets(ctx context.Context, ticker string, limit int) ([]BalanceSheet, error) {
	var result []BalanceSheet
	err := c.Fetch(ctx, "balance-sheet-statement", P{
		"symbol": ticker,
		"limit":  fmt.Sprintf("%d", limit),
		"period": "annual",
	}, &result)
	return result, err
}

func (c *Client) GetCashFlowStatements(ctx context.Context, ticker string, limit int) ([]CashFlowStatement, error) {
	var result []CashFlowStatement
	err := c.Fetch(ctx, "cash-flow-statement", P{
		"symbol": ticker,
		"limit":  fmt.Sprintf("%d", limit),
		"period": "annual",
	}, &result)
	return result, err
}

func (c *Client) GetQuote(ctx context.Context, ticker string) (*Quote, error) {
	var result []Quote
	if err := c.Fetch(ctx, "quote", P{"symbol": ticker}, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("fmp: no quote data for %s", ticker)
	}
	return &result[0], nil
}

// GetQuotes fetches quotes for multiple tickers in a single API call.
func (c *Client) GetQuotes(ctx context.Context, tickers []string) ([]Quote, error) {
	var result []Quote
	err := c.Fetch(ctx, "quote", P{"symbol": strings.Join(tickers, ",")}, &result)
	return result, err
}

// InsiderTransaction represents one Form 4 SEC filing transaction.
type InsiderTransaction struct {
	Symbol                  string  `json:"symbol"`
	FilingDate              string  `json:"filingDate"`
	TransactionDate         string  `json:"transactionDate"`
	ReportingName           string  `json:"reportingName"`
	TypeOfOwner             string  `json:"typeOfOwner"`
	AcquistionOrDisposition string  `json:"acquistionOrDisposition"`
	SecuritiesTransacted    float64 `json:"securitiesTransacted"`
	Price                   float64 `json:"price"`
	SecuritiesOwned         float64 `json:"securitiesOwned"`
	SecurityName            string  `json:"securityName"`
	TransactionType         string  `json:"transactionType"`
}

func (c *Client) GetInsiderTrading(ctx context.Context, ticker string, limit int) ([]InsiderTransaction, error) {
	// insider-trading is only available in v3 (not in stable)
	u := v3Base + "/insider-trading?symbol=" + url.QueryEscape(ticker) + "&apikey=" + url.QueryEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fmp: insider-trading returned status %d", resp.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("fmp: insider-trading invalid JSON: %w", err)
	}
	if len(raw) > 0 && raw[0] == '{' {
		var errResp struct {
			ErrorMessage string `json:"Error Message"`
			Message      string `json:"message"`
		}
		if json.Unmarshal(raw, &errResp) == nil {
			if errResp.ErrorMessage != "" {
				return nil, fmt.Errorf("fmp: insider-trading: %s", errResp.ErrorMessage)
			}
			if errResp.Message != "" {
				return nil, fmt.Errorf("fmp: insider-trading: %s", errResp.Message)
			}
		}
	}
	var result []InsiderTransaction
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("fmp: insider-trading decode: %w", err)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
