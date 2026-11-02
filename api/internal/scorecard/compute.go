// Package scorecard computes financial health metrics from raw statement data.
package scorecard

import (
	"fmt"
	"math"

	"github.com/tripleboys27/stock-dashboard/api/internal/fmp"
)

// Scorecard holds all computed financial metrics for one period.
type Scorecard struct {
	Ticker    string `json:"ticker"`
	PeriodEnd string `json:"period_end"`
	Period    string `json:"period"` // "FY", "Q1", etc.

	// === PROFITABILITAS ===
	Revenue        float64  `json:"revenue"`
	RevenueGrowth  *float64 `json:"revenue_growth,omitempty"`  // YoY %
	GrossMargin    *float64 `json:"gross_margin,omitempty"`    // %
	OperatingMargin *float64 `json:"operating_margin,omitempty"` // %
	EBITDAMargin   *float64 `json:"ebitda_margin,omitempty"`   // %
	NetMargin      *float64 `json:"net_margin,omitempty"`      // %
	EPS            float64  `json:"eps"`
	EPSGrowth      *float64 `json:"eps_growth,omitempty"` // YoY %
	ROE            *float64 `json:"roe,omitempty"`        // Net Income / Equity
	ROA            *float64 `json:"roa,omitempty"`        // Net Income / Assets
	ROIC           *float64 `json:"roic,omitempty"`       // EBIT(1-t) / Invested Capital

	// === LIKUIDITAS ===
	CurrentRatio *float64 `json:"current_ratio,omitempty"` // >1.5 aman
	QuickRatio   *float64 `json:"quick_ratio,omitempty"`   // >1 aman
	CashPosition float64  `json:"cash_position"`           // absolute

	// === SOLVABILITAS / LEVERAGE ===
	DebtToEquity     *float64 `json:"debt_to_equity,omitempty"`      // <1 konservatif
	NetDebt          float64  `json:"net_debt"`                      // bisa negatif (net cash)
	NetDebtToEBITDA  *float64 `json:"net_debt_to_ebitda,omitempty"`  // <3x aman
	InterestCoverage *float64 `json:"interest_coverage,omitempty"`   // >3x aman

	// === ARUS KAS ===
	OperatingCashFlow float64  `json:"operating_cash_flow"`
	CapEx             float64  `json:"capex"`                   // negatif = pengeluaran
	FreeCashFlow      float64  `json:"free_cash_flow"`
	FCFMargin         *float64 `json:"fcf_margin,omitempty"`    // %
	FCFConversion     *float64 `json:"fcf_conversion,omitempty"` // FCF / Net Income

	// === VALUASI (butuh harga saham) ===
	Price     float64  `json:"price,omitempty"`
	MarketCap float64  `json:"market_cap,omitempty"`
	PERatio   *float64 `json:"pe_ratio,omitempty"`   // Price / EPS
	PSRatio   *float64 `json:"ps_ratio,omitempty"`   // MarketCap / Revenue
	PBRatio   *float64 `json:"pb_ratio,omitempty"`   // Price / Book Value per share
	EVEBITDARatio *float64 `json:"ev_ebitda,omitempty"` // EV / EBITDA

	// === SKOR KUALITAS ===
	PiotroskiScore int    `json:"piotroski_score"` // 0-9; ≥7 sehat
	PiotroskiGrade string `json:"piotroski_grade"` // "Sehat", "Netral", "Lemah"

	// === FLAG / PERINGATAN ===
	Flags []Flag `json:"flags"`
}

// Flag adalah peringatan atau highlight untuk investor.
type Flag struct {
	Level   string `json:"level"`   // "ok", "warning", "danger"
	Metric  string `json:"metric"`
	Message string `json:"message"`
}

// Compute builds a Scorecard from raw FMP data.
// prev is the previous period (used for YoY growth); pass nil if unavailable.
func Compute(
	ticker string,
	income fmp.IncomeStatement,
	balance fmp.BalanceSheet,
	cf fmp.CashFlowStatement,
	quote *fmp.Quote, // may be nil
	prev *fmp.IncomeStatement, // previous period for growth calc; may be nil
	prevBalance *fmp.BalanceSheet, // for avg equity/assets; may be nil
) Scorecard {
	s := Scorecard{
		Ticker:            ticker,
		PeriodEnd:         income.Date,
		Period:            income.Period,
		Revenue:           income.Revenue,
		EPS:               income.EPSDiluted,
		CashPosition:      balance.CashAndEquivalents,
		OperatingCashFlow: cf.OperatingCashFlow,
		CapEx:             cf.CapitalExpenditure,
		FreeCashFlow:      cf.FreeCashFlow,
		NetDebt:           (balance.LongTermDebt + balance.ShortTermDebt) - balance.CashAndEquivalents,
	}

	// ── Profitabilitas ─────────────────────────────────────────────────────────
	if income.Revenue != 0 {
		gm := income.GrossProfit / income.Revenue
		s.GrossMargin = &gm
		om := income.OperatingIncome / income.Revenue
		s.OperatingMargin = &om
		nm := income.NetIncome / income.Revenue
		s.NetMargin = &nm
		if income.EBITDA != 0 {
			em := income.EBITDA / income.Revenue
			s.EBITDAMargin = &em
		}
		fcfm := cf.FreeCashFlow / income.Revenue
		s.FCFMargin = &fcfm
	}

	if prev != nil && prev.Revenue != 0 {
		rg := (income.Revenue - prev.Revenue) / math.Abs(prev.Revenue)
		s.RevenueGrowth = &rg
	}
	if prev != nil && prev.EPSDiluted != 0 {
		eg := (income.EPSDiluted - prev.EPSDiluted) / math.Abs(prev.EPSDiluted)
		s.EPSGrowth = &eg
	}

	// Average equity and assets for ROE/ROA
	avgEquity := balance.TotalStockholdersEquity
	avgAssets := balance.TotalAssets
	if prevBalance != nil {
		avgEquity = (balance.TotalStockholdersEquity + prevBalance.TotalStockholdersEquity) / 2
		avgAssets = (balance.TotalAssets + prevBalance.TotalAssets) / 2
	}
	if avgEquity != 0 {
		roe := income.NetIncome / avgEquity
		s.ROE = &roe
	}
	if avgAssets != 0 {
		roa := income.NetIncome / avgAssets
		s.ROA = &roa
	}

	// ROIC = EBIT*(1-effectiveTaxRate) / (TotalDebt + Equity - Cash)
	investedCapital := balance.TotalDebt + balance.TotalStockholdersEquity - balance.CashAndEquivalents
	if investedCapital != 0 && income.OperatingIncome != 0 {
		taxRate := 0.21 // default US corporate tax rate
		if income.NetIncome != 0 && income.IncomeTaxExpense != 0 {
			preTax := income.NetIncome + income.IncomeTaxExpense
			if preTax != 0 {
				taxRate = income.IncomeTaxExpense / preTax
				if taxRate < 0 {
					taxRate = 0
				}
				if taxRate > 0.5 {
					taxRate = 0.5
				}
			}
		}
		roic := income.OperatingIncome * (1 - taxRate) / investedCapital
		s.ROIC = &roic
	}

	// FCF Conversion = FCF / Net Income
	if income.NetIncome != 0 {
		conv := cf.FreeCashFlow / income.NetIncome
		s.FCFConversion = &conv
	}

	// ── Likuiditas ─────────────────────────────────────────────────────────────
	if balance.TotalCurrentLiabilities != 0 {
		cr := balance.TotalCurrentAssets / balance.TotalCurrentLiabilities
		s.CurrentRatio = &cr
		qr := (balance.TotalCurrentAssets - balance.Inventory) / balance.TotalCurrentLiabilities
		s.QuickRatio = &qr
	}

	// ── Leverage ───────────────────────────────────────────────────────────────
	if balance.TotalStockholdersEquity != 0 {
		de := balance.TotalDebt / balance.TotalStockholdersEquity
		s.DebtToEquity = &de
	}
	if income.EBITDA != 0 {
		nd := s.NetDebt / math.Abs(income.EBITDA)
		s.NetDebtToEBITDA = &nd
	}
	// Interest coverage: EBIT / |Interest Expense|
	interestAbs := math.Abs(income.InterestExpense)
	if interestAbs > 0 {
		ic := income.OperatingIncome / interestAbs
		s.InterestCoverage = &ic
	}

	// ── Valuasi ────────────────────────────────────────────────────────────────
	if quote != nil {
		s.Price = quote.Price
		s.MarketCap = quote.MarketCap
		if income.EPSDiluted != 0 && quote.Price != 0 {
			pe := quote.Price / income.EPSDiluted
			if pe > 0 {
				s.PERatio = &pe
			}
		}
		if income.Revenue != 0 && quote.MarketCap != 0 {
			ps := quote.MarketCap / income.Revenue
			s.PSRatio = &ps
		}
		bookValuePerShare := 0.0
		if quote.SharesOutstanding != 0 {
			bookValuePerShare = balance.TotalStockholdersEquity / quote.SharesOutstanding
		}
		if bookValuePerShare > 0 && quote.Price != 0 {
			pb := quote.Price / bookValuePerShare
			s.PBRatio = &pb
		}
		// EV = MarketCap + TotalDebt - Cash
		if income.EBITDA != 0 {
			ev := quote.MarketCap + balance.TotalDebt - balance.CashAndEquivalents
			evEbitda := ev / income.EBITDA
			if evEbitda > 0 {
				s.EVEBITDARatio = &evEbitda
			}
		}
	}

	// ── Piotroski F-Score (9 poin) ─────────────────────────────────────────────
	s.PiotroskiScore = piotroski(income, balance, cf, prev, prevBalance)
	switch {
	case s.PiotroskiScore >= 7:
		s.PiotroskiGrade = "Sehat"
	case s.PiotroskiScore >= 4:
		s.PiotroskiGrade = "Netral"
	default:
		s.PiotroskiGrade = "Lemah"
	}

	// ── Flags ─────────────────────────────────────────────────────────────────
	s.Flags = buildFlags(s)

	return s
}

// piotroski computes the Piotroski F-Score (0–9).
// Each of the 9 criteria contributes 1 point.
func piotroski(
	income fmp.IncomeStatement,
	balance fmp.BalanceSheet,
	cf fmp.CashFlowStatement,
	prev *fmp.IncomeStatement,
	prevBalance *fmp.BalanceSheet,
) int {
	score := 0

	// Profitability signals (4 points)
	if income.NetIncome > 0 {
		score++ // F1: positive net income
	}
	if balance.TotalAssets > 0 && income.NetIncome/balance.TotalAssets > 0 {
		score++ // F2: positive ROA
	}
	if cf.OperatingCashFlow > 0 {
		score++ // F3: positive operating cash flow
	}
	if cf.OperatingCashFlow > income.NetIncome {
		score++ // F4: cash flow > net income (quality earnings)
	}

	// Leverage & liquidity signals (3 points)
	if prevBalance != nil {
		prevLeverage := 0.0
		currLeverage := 0.0
		if prevBalance.TotalAssets != 0 {
			prevLeverage = (prevBalance.LongTermDebt) / prevBalance.TotalAssets
		}
		if balance.TotalAssets != 0 {
			currLeverage = (balance.LongTermDebt) / balance.TotalAssets
		}
		if currLeverage <= prevLeverage {
			score++ // F5: leverage not increasing
		}

		prevCR := 0.0
		currCR := 0.0
		if prevBalance.TotalCurrentLiabilities != 0 {
			prevCR = prevBalance.TotalCurrentAssets / prevBalance.TotalCurrentLiabilities
		}
		if balance.TotalCurrentLiabilities != 0 {
			currCR = balance.TotalCurrentAssets / balance.TotalCurrentLiabilities
		}
		if currCR >= prevCR {
			score++ // F6: liquidity improving
		}
	}
	// F7: no dilution (shares not increased significantly) — simplified: skip if no prev data
	if prev == nil || income.WeightedAvgSharesOut <= prev.WeightedAvgSharesOut*1.02 {
		score++ // ≤2% share increase
	}

	// Operating efficiency signals (2 points)
	if prevBalance != nil && prev != nil {
		prevGM := 0.0
		if prev.Revenue != 0 {
			prevGM = prev.GrossProfit / prev.Revenue
		}
		currGM := 0.0
		if income.Revenue != 0 {
			currGM = income.GrossProfit / income.Revenue
		}
		if currGM >= prevGM {
			score++ // F8: gross margin improving
		}

		prevATO := 0.0
		if prevBalance.TotalAssets != 0 {
			prevATO = prev.Revenue / prevBalance.TotalAssets
		}
		currATO := 0.0
		if balance.TotalAssets != 0 {
			currATO = income.Revenue / balance.TotalAssets
		}
		if currATO >= prevATO {
			score++ // F9: asset turnover improving
		}
	}

	return score
}

// buildFlags generates human-readable warnings and highlights.
func buildFlags(s Scorecard) []Flag {
	var flags []Flag

	add := func(level, metric, msg string) {
		flags = append(flags, Flag{Level: level, Metric: metric, Message: msg})
	}

	// Net margin
	if s.NetMargin != nil {
		switch {
		case *s.NetMargin < 0:
			add("danger", "net_margin", fmt.Sprintf("Margin laba negatif (%.1f%%) — perusahaan merugi", *s.NetMargin*100))
		case *s.NetMargin < 0.05:
			add("warning", "net_margin", fmt.Sprintf("Margin laba tipis (%.1f%%) — rentan terhadap kenaikan biaya", *s.NetMargin*100))
		case *s.NetMargin > 0.20:
			add("ok", "net_margin", fmt.Sprintf("Margin laba sangat baik (%.1f%%)", *s.NetMargin*100))
		}
	}

	// Current ratio
	if s.CurrentRatio != nil {
		switch {
		case *s.CurrentRatio < 1.0:
			add("danger", "current_ratio", fmt.Sprintf("Current ratio %.2fx — liabilitas jangka pendek melebihi aset lancar, risiko likuiditas tinggi", *s.CurrentRatio))
		case *s.CurrentRatio < 1.5:
			add("warning", "current_ratio", fmt.Sprintf("Current ratio %.2fx — likuiditas cukup namun perlu dipantau", *s.CurrentRatio))
		default:
			add("ok", "current_ratio", fmt.Sprintf("Current ratio %.2fx — likuiditas sehat", *s.CurrentRatio))
		}
	}

	// Net Debt/EBITDA
	if s.NetDebtToEBITDA != nil {
		switch {
		case *s.NetDebtToEBITDA < 0:
			add("ok", "net_debt", fmt.Sprintf("Posisi net cash (Net Debt/EBITDA %.1fx) — tidak ada beban utang bersih", *s.NetDebtToEBITDA))
		case *s.NetDebtToEBITDA > 4:
			add("danger", "net_debt", fmt.Sprintf("Leverage sangat tinggi (Net Debt/EBITDA %.1fx) — beban utang berat", *s.NetDebtToEBITDA))
		case *s.NetDebtToEBITDA > 2.5:
			add("warning", "net_debt", fmt.Sprintf("Leverage moderat-tinggi (Net Debt/EBITDA %.1fx) — perlu diperhatikan", *s.NetDebtToEBITDA))
		default:
			add("ok", "net_debt", fmt.Sprintf("Leverage aman (Net Debt/EBITDA %.1fx)", *s.NetDebtToEBITDA))
		}
	}

	// Interest coverage
	if s.InterestCoverage != nil {
		switch {
		case *s.InterestCoverage < 1.5:
			add("danger", "interest_coverage", fmt.Sprintf("Interest coverage %.1fx — laba operasional hampir tidak cukup bayar bunga", *s.InterestCoverage))
		case *s.InterestCoverage < 3:
			add("warning", "interest_coverage", fmt.Sprintf("Interest coverage %.1fx — kemampuan bayar bunga terbatas", *s.InterestCoverage))
		case *s.InterestCoverage > 10:
			add("ok", "interest_coverage", fmt.Sprintf("Interest coverage %.1fx — sangat nyaman membayar bunga", *s.InterestCoverage))
		}
	}

	// FCF conversion
	if s.FCFConversion != nil {
		switch {
		case *s.FCFConversion < 0:
			add("warning", "fcf_conversion", fmt.Sprintf("Free cash flow negatif (%.0f%% dari net income) — perlu cermati kebutuhan capex", *s.FCFConversion*100))
		case *s.FCFConversion < 0.5:
			add("warning", "fcf_conversion", fmt.Sprintf("Konversi FCF rendah (%.0f%% dari net income) — sebagian besar laba tidak menjadi kas nyata", *s.FCFConversion*100))
		case *s.FCFConversion > 1.0:
			add("ok", "fcf_conversion", fmt.Sprintf("Konversi FCF sangat baik (%.0f%% dari net income) — laba berkualitas tinggi", *s.FCFConversion*100))
		}
	}

	// ROE
	if s.ROE != nil {
		switch {
		case *s.ROE > 0.20:
			add("ok", "roe", fmt.Sprintf("ROE %.1f%% — sangat efisien menggunakan modal pemegang saham", *s.ROE*100))
		case *s.ROE < 0:
			add("danger", "roe", fmt.Sprintf("ROE negatif (%.1f%%) — ekuitas tidak menghasilkan imbal hasil", *s.ROE*100))
		}
	}

	// Revenue growth
	if s.RevenueGrowth != nil {
		switch {
		case *s.RevenueGrowth < -0.10:
			add("danger", "revenue_growth", fmt.Sprintf("Pendapatan turun %.1f%% YoY — perlambatan bisnis signifikan", *s.RevenueGrowth*100))
		case *s.RevenueGrowth < 0:
			add("warning", "revenue_growth", fmt.Sprintf("Pendapatan turun %.1f%% YoY", *s.RevenueGrowth*100))
		case *s.RevenueGrowth > 0.20:
			add("ok", "revenue_growth", fmt.Sprintf("Pertumbuhan pendapatan kuat +%.1f%% YoY", *s.RevenueGrowth*100))
		}
	}

	// Piotroski
	switch {
	case s.PiotroskiScore >= 7:
		add("ok", "piotroski", fmt.Sprintf("Piotroski F-Score %d/9 — fundamental kuat secara keseluruhan", s.PiotroskiScore))
	case s.PiotroskiScore <= 2:
		add("danger", "piotroski", fmt.Sprintf("Piotroski F-Score %d/9 — banyak sinyal fundamental yang lemah", s.PiotroskiScore))
	}

	return flags
}

