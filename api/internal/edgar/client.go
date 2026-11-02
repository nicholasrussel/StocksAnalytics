// Package edgar fetches Form 4 insider trading data from SEC EDGAR (free, no API key needed).
// SEC requires a User-Agent header identifying the app: "AppName contact@email.com"
package edgar

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	dataBase    = "https://data.sec.gov"
	archiveBase = "https://www.sec.gov/Archives/edgar/data"
	tickersURL  = "https://www.sec.gov/files/company_tickers_exchange.json"
)

type Client struct {
	http      *http.Client
	userAgent string
	cikCache  map[string]cikEntry
}

type cikEntry struct {
	padded  string // "0000320193"
	numeric string // "320193"
}

func NewClient(userAgent string) *Client {
	return &Client{
		http:      &http.Client{Timeout: 20 * time.Second},
		userAgent: userAgent,
		cikCache:  make(map[string]cikEntry),
	}
}

type InsiderTransaction struct {
	ReporterName    string  `json:"reporter_name"`
	OfficerTitle    string  `json:"officer_title"`
	IsDirector      bool    `json:"is_director"`
	IsOfficer       bool    `json:"is_officer"`
	TransactionDate string  `json:"transaction_date"`
	FilingDate      string  `json:"filing_date"`
	TransactionCode string  `json:"transaction_code"` // P=buy, S=sell, A=award, F=tax, M=option, G=gift
	Shares          float64 `json:"shares"`
	PricePerShare   float64 `json:"price_per_share"`
	SharesOwned     float64 `json:"shares_owned"`
	AcqDisp         string  `json:"acq_disp"` // A or D
	SecurityTitle   string  `json:"security_title"`
}

// ── HTTP helper ────────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	return c.http.Do(req)
}

// ── CIK lookup ─────────────────────────────────────────────────────────────────

type tickersExchange struct {
	Fields []string `json:"fields"` // ["cik","name","ticker","exchange"]
	Data   [][]any  `json:"data"`
}

func (c *Client) lookupCIK(ctx context.Context, ticker string) (cikEntry, error) {
	upper := strings.ToUpper(ticker)
	if v, ok := c.cikCache[upper]; ok {
		return v, nil
	}

	resp, err := c.get(ctx, tickersURL)
	if err != nil {
		return cikEntry{}, fmt.Errorf("edgar: lookup CIK: %w", err)
	}
	defer resp.Body.Close()

	var data tickersExchange
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return cikEntry{}, fmt.Errorf("edgar: decode tickers: %w", err)
	}

	cikIdx, tickerIdx := -1, -1
	for i, f := range data.Fields {
		switch f {
		case "cik":
			cikIdx = i
		case "ticker":
			tickerIdx = i
		}
	}
	if cikIdx < 0 || tickerIdx < 0 {
		return cikEntry{}, fmt.Errorf("edgar: unexpected tickers schema")
	}

	for _, row := range data.Data {
		if len(row) <= cikIdx || len(row) <= tickerIdx {
			continue
		}
		t, _ := row[tickerIdx].(string)
		if !strings.EqualFold(t, ticker) {
			continue
		}
		var n int64
		switch v := row[cikIdx].(type) {
		case float64:
			n = int64(v)
		default:
			continue
		}
		e := cikEntry{
			padded:  fmt.Sprintf("%010d", n),
			numeric: fmt.Sprintf("%d", n),
		}
		c.cikCache[upper] = e
		return e, nil
	}
	return cikEntry{}, fmt.Errorf("edgar: CIK not found for %s", ticker)
}

// ── Submissions ────────────────────────────────────────────────────────────────

type submissionsResp struct {
	Filings struct {
		Recent struct {
			Form            []string `json:"form"`
			FilingDate      []string `json:"filingDate"`
			AccessionNumber []string `json:"accessionNumber"`
			PrimaryDocument []string `json:"primaryDocument"`
			IsInlineXBRL    []int    `json:"isInlineXBRL"`
		} `json:"recent"`
	} `json:"filings"`
}

type filingRef struct {
	accession  string
	filingDate string
	primaryDoc string
	inlineXBRL bool
}

func (c *Client) getForm4Refs(ctx context.Context, cik cikEntry, limit int) ([]filingRef, error) {
	url := fmt.Sprintf("%s/submissions/CIK%s.json", dataBase, cik.padded)
	resp, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("edgar: submissions status %d", resp.StatusCode)
	}

	var subs submissionsResp
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("edgar: decode submissions: %w", err)
	}

	r := subs.Filings.Recent
	var refs []filingRef
	for i, form := range r.Form {
		if form != "4" && form != "4/A" {
			continue
		}
		inline := len(r.IsInlineXBRL) > i && r.IsInlineXBRL[i] == 1
		refs = append(refs, filingRef{
			accession:  r.AccessionNumber[i],
			filingDate: r.FilingDate[i],
			primaryDoc: r.PrimaryDocument[i],
			inlineXBRL: inline,
		})
		if limit > 0 && len(refs) >= limit {
			break
		}
	}
	return refs, nil
}

// ── Form 4 XML parsing ─────────────────────────────────────────────────────────

type ownershipDoc struct {
	Owner struct {
		ID struct {
			Name string `xml:"rptOwnerName"`
		} `xml:"reportingOwnerId"`
		Rel struct {
			IsDirector   string `xml:"isDirector"`
			IsOfficer    string `xml:"isOfficer"`
			OfficerTitle string `xml:"officerTitle"`
		} `xml:"reportingOwnerRelationship"`
	} `xml:"reportingOwner"`
	NonDeriv struct {
		Transactions []ndTx `xml:"nonDerivativeTransaction"`
	} `xml:"nonDerivativeTable"`
}

type ndTx struct {
	SecurityTitle struct {
		Value string `xml:"value"`
	} `xml:"securityTitle"`
	TxDate struct {
		Value string `xml:"value"`
	} `xml:"transactionDate"`
	Coding struct {
		Code string `xml:"transactionCode"`
	} `xml:"transactionCoding"`
	Amounts struct {
		Shares  struct {
			Value float64 `xml:"value"`
		} `xml:"transactionShares"`
		Price struct {
			Value float64 `xml:"value"`
		} `xml:"transactionPricePerShare"`
		AcqDisp struct {
			Value string `xml:"value"`
		} `xml:"transactionAcquiredDisposedCode"`
	} `xml:"transactionAmounts"`
	Post struct {
		Owned struct {
			Value float64 `xml:"value"`
		} `xml:"sharesOwnedFollowingTransaction"`
	} `xml:"postTransactionAmounts"`
}

func isTruthy(s string) bool {
	return s == "1" || strings.EqualFold(s, "true")
}

func (c *Client) fetchXML(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) parseForm4(ctx context.Context, cik cikEntry, ref filingRef) ([]InsiderTransaction, error) {
	accNoDash := strings.ReplaceAll(ref.accession, "-", "")
	baseName := strings.TrimSuffix(strings.TrimSuffix(ref.primaryDoc, ".htm"), ".html")

	// Build candidate XML URLs in priority order
	candidates := []string{}
	if strings.HasSuffix(ref.primaryDoc, ".xml") {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s",
			archiveBase, cik.numeric, accNoDash, ref.primaryDoc))
	} else if ref.inlineXBRL {
		// iXBRL companion has _htm.xml suffix
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s_htm.xml",
			archiveBase, cik.numeric, accNoDash, baseName))
	}
	// Legacy fallbacks
	candidates = append(candidates,
		fmt.Sprintf("%s/%s/%s/form4.xml", archiveBase, cik.numeric, accNoDash),
		fmt.Sprintf("%s/%s/%s/doc4.xml", archiveBase, cik.numeric, accNoDash),
	)

	var xmlBytes []byte
	for _, u := range candidates {
		b, err := c.fetchXML(ctx, u)
		if err == nil {
			xmlBytes = b
			break
		}
	}
	if xmlBytes == nil {
		return nil, fmt.Errorf("edgar: no XML found for %s", ref.accession)
	}

	var doc ownershipDoc
	if err := xml.Unmarshal(xmlBytes, &doc); err != nil {
		return nil, fmt.Errorf("edgar: parse XML %s: %w", ref.accession, err)
	}

	name := strings.TrimSpace(doc.Owner.ID.Name)
	title := strings.TrimSpace(doc.Owner.Rel.OfficerTitle)
	isDir := isTruthy(doc.Owner.Rel.IsDirector)
	isOff := isTruthy(doc.Owner.Rel.IsOfficer)

	var txns []InsiderTransaction
	for _, tx := range doc.NonDeriv.Transactions {
		txns = append(txns, InsiderTransaction{
			ReporterName:    name,
			OfficerTitle:    title,
			IsDirector:      isDir,
			IsOfficer:       isOff,
			TransactionDate: tx.TxDate.Value,
			FilingDate:      ref.filingDate,
			TransactionCode: tx.Coding.Code,
			Shares:          tx.Amounts.Shares.Value,
			PricePerShare:   tx.Amounts.Price.Value,
			SharesOwned:     tx.Post.Owned.Value,
			AcqDisp:         tx.Amounts.AcqDisp.Value,
			SecurityTitle:   tx.SecurityTitle.Value,
		})
	}
	return txns, nil
}

// ── Public API ─────────────────────────────────────────────────────────────────

// GetInsiderTransactions fetches recent Form 4 insider transactions from SEC EDGAR.
// limit controls how many Form 4 filings to parse (each may have multiple transactions).
func (c *Client) GetInsiderTransactions(ctx context.Context, ticker string, filingLimit int) ([]InsiderTransaction, error) {
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return nil, err
	}

	refs, err := c.getForm4Refs(ctx, cik, filingLimit)
	if err != nil {
		return nil, err
	}

	var all []InsiderTransaction
	for _, ref := range refs {
		// SEC rate limit: max 10 req/sec; 200ms keeps us well under
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		txns, err := c.parseForm4(ctx, cik, ref)
		if err != nil {
			continue // skip unparseable filings
		}
		all = append(all, txns...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].TransactionDate > all[j].TransactionDate
	})
	return all, nil
}
