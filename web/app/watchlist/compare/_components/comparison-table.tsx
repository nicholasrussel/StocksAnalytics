import Link from "next/link";
import type { CompareEntry, Scorecard } from "../page";

// ── Helpers ────────────────────────────────────────────────────────────────────

function pct(v?: number) {
  if (v == null) return null;
  return (v * 100).toFixed(1) + "%";
}
function num(v?: number, dec = 2) {
  if (v == null) return null;
  if (Math.abs(v) >= 1e12) return (v / 1e12).toFixed(1) + "T";
  if (Math.abs(v) >= 1e9) return (v / 1e9).toFixed(1) + "B";
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(1) + "M";
  return v.toFixed(dec);
}
function ratio(v?: number) {
  if (v == null) return null;
  return v.toFixed(2) + "x";
}

type Direction = "higher" | "lower" | "none";

interface MetricDef {
  label: string;
  get: (s: Scorecard) => number | null | undefined;
  fmt: (v: number) => string;
  dir: Direction;
  prefix?: string;
}

const SECTIONS: { title: string; metrics: MetricDef[] }[] = [
  {
    title: "Profitabilitas",
    metrics: [
      { label: "Pendapatan", get: s => s.revenue, fmt: v => `$${num(v, 0)!}`, dir: "higher" },
      { label: "Revenue Growth", get: s => s.revenue_growth, fmt: v => pct(v)!, dir: "higher" },
      { label: "Gross Margin", get: s => s.gross_margin, fmt: v => pct(v)!, dir: "higher" },
      { label: "Operating Margin", get: s => s.operating_margin, fmt: v => pct(v)!, dir: "higher" },
      { label: "Net Margin", get: s => s.net_margin, fmt: v => pct(v)!, dir: "higher" },
      { label: "EPS", get: s => s.eps, fmt: v => `$${num(v)!}`, dir: "higher" },
      { label: "ROE", get: s => s.roe, fmt: v => pct(v)!, dir: "higher" },
      { label: "ROA", get: s => s.roa, fmt: v => pct(v)!, dir: "higher" },
      { label: "ROIC", get: s => s.roic, fmt: v => pct(v)!, dir: "higher" },
    ],
  },
  {
    title: "Likuiditas",
    metrics: [
      { label: "Current Ratio", get: s => s.current_ratio, fmt: v => ratio(v)!, dir: "higher" },
      { label: "Quick Ratio", get: s => s.quick_ratio, fmt: v => ratio(v)!, dir: "higher" },
      { label: "Kas & Setara", get: s => s.cash_position, fmt: v => `$${num(v, 0)!}`, dir: "higher" },
    ],
  },
  {
    title: "Leverage",
    metrics: [
      { label: "Debt / Equity", get: s => s.debt_to_equity, fmt: v => ratio(v)!, dir: "lower" },
      { label: "Net Debt / EBITDA", get: s => s.net_debt_to_ebitda, fmt: v => ratio(v)!, dir: "lower" },
      { label: "Interest Coverage", get: s => s.interest_coverage, fmt: v => ratio(v)!, dir: "higher" },
    ],
  },
  {
    title: "Arus Kas",
    metrics: [
      { label: "Free Cash Flow", get: s => s.free_cash_flow, fmt: v => `$${num(v, 0)!}`, dir: "higher" },
      { label: "FCF Margin", get: s => s.fcf_margin, fmt: v => pct(v)!, dir: "higher" },
      { label: "FCF Conversion", get: s => s.fcf_conversion, fmt: v => (v * 100).toFixed(0) + "%", dir: "higher" },
    ],
  },
  {
    title: "Valuasi",
    metrics: [
      { label: "Harga Saham", get: s => s.price, fmt: v => `$${v.toFixed(2)}`, dir: "none" },
      { label: "P/E", get: s => s.pe_ratio, fmt: v => ratio(v)!, dir: "none" },
      { label: "P/S", get: s => s.ps_ratio, fmt: v => ratio(v)!, dir: "none" },
      { label: "P/B", get: s => s.pb_ratio, fmt: v => ratio(v)!, dir: "none" },
      { label: "EV / EBITDA", get: s => s.ev_ebitda, fmt: v => ratio(v)!, dir: "none" },
    ],
  },
];

// ── Color logic ────────────────────────────────────────────────────────────────

function rankValues(values: (number | null)[], dir: Direction): Map<number, "best" | "worst"> {
  const result = new Map<number, "best" | "worst">();
  if (dir === "none") return result;

  const valid = values.filter((v): v is number => v != null);
  if (valid.length < 2) return result;

  const max = Math.max(...valid);
  const min = Math.min(...valid);
  if (max === min) return result;

  const bestVal = dir === "higher" ? max : min;
  const worstVal = dir === "higher" ? min : max;

  for (let i = 0; i < values.length; i++) {
    const v = values[i];
    if (v === null) continue;
    if (v === bestVal) result.set(i, "best");
    else if (v === worstVal) result.set(i, "worst");
  }
  return result;
}

function rankClass(rank: "best" | "worst" | undefined) {
  if (rank === "best") return "text-green-600 dark:text-green-400 font-semibold";
  if (rank === "worst") return "text-red-500 dark:text-red-400";
  return "text-zinc-700 dark:text-zinc-300";
}

// ── Piotroski badge ────────────────────────────────────────────────────────────

function piotroskiBadge(score: number, grade: string) {
  const color =
    grade === "Sehat"
      ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
      : grade === "Lemah"
      ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
      : "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400";
  return (
    <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded-full ${color}`}>
      {score}/9 {grade}
    </span>
  );
}

// ── Main ───────────────────────────────────────────────────────────────────────

export function ComparisonTable({ entries }: { entries: CompareEntry[] }) {
  const cols = entries.length;

  return (
    <div className="overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
      <table className="w-full text-sm border-collapse">
        {/* Header */}
        <thead>
          <tr className="border-b border-zinc-200 dark:border-zinc-800">
            <th className="text-left px-4 py-3 text-xs font-semibold text-zinc-400 dark:text-zinc-500 uppercase tracking-wide w-36">
              Metrik
            </th>
            {entries.map((e) => (
              <th key={e.ticker} className="px-4 py-3 text-center min-w-[120px]">
                <div className="flex flex-col items-center gap-1">
                  <Link
                    href={`/watchlist/${e.ticker}`}
                    className="text-base font-bold text-zinc-900 dark:text-zinc-50 hover:text-blue-600 dark:hover:text-blue-400"
                  >
                    {e.ticker}
                  </Link>
                  {e.scorecard ? (
                    <>
                      <span className="text-[10px] text-zinc-400 dark:text-zinc-500">
                        {e.scorecard.period} {e.period_end?.slice(0, 4)}
                      </span>
                      {piotroskiBadge(e.scorecard.piotroski_score, e.scorecard.piotroski_grade)}
                    </>
                  ) : (
                    <span className="text-[10px] text-amber-500">Data belum tersedia</span>
                  )}
                </div>
              </th>
            ))}
          </tr>
        </thead>

        {/* Sections */}
        <tbody>
          {SECTIONS.map((section) => (
            <>
              {/* Section header */}
              <tr key={`sec-${section.title}`} className="bg-zinc-50 dark:bg-zinc-800/50">
                <td
                  colSpan={1 + cols}
                  className="px-4 py-1.5 text-[10px] font-semibold text-zinc-400 dark:text-zinc-600 uppercase tracking-widest"
                >
                  {section.title}
                </td>
              </tr>

              {/* Metric rows */}
              {section.metrics.map((metric) => {
                const rawValues = entries.map((e) =>
                  e.scorecard ? (metric.get(e.scorecard) ?? null) : null
                );
                const ranks = rankValues(rawValues, metric.dir);

                // Skip rows where all values are null
                if (rawValues.every((v) => v === null)) return null;

                return (
                  <tr
                    key={metric.label}
                    className="border-t border-zinc-100 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-800/30 transition-colors"
                  >
                    <td className="px-4 py-2 text-xs text-zinc-500 dark:text-zinc-400 whitespace-nowrap">
                      {metric.label}
                    </td>
                    {entries.map((e, i) => {
                      const raw = rawValues[i];
                      const rank = ranks.get(i);
                      return (
                        <td key={e.ticker} className="px-4 py-2 text-center">
                          {!e.scorecard ? (
                            <span className="text-xs text-zinc-300 dark:text-zinc-600">—</span>
                          ) : raw === null ? (
                            <span className="text-xs text-zinc-300 dark:text-zinc-600">—</span>
                          ) : (
                            <span className={`text-xs ${rankClass(rank)}`}>
                              {metric.fmt(raw)}
                            </span>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
            </>
          ))}
        </tbody>
      </table>

      <div className="px-4 py-3 border-t border-zinc-100 dark:border-zinc-800 text-xs text-zinc-400 dark:text-zinc-600">
        <span className="text-green-600 dark:text-green-400 font-semibold">Hijau</span> = terbaik ·{" "}
        <span className="text-red-500 dark:text-red-400">Merah</span> = terburuk di baris ini ·
        Valuasi tidak di-rank (konteks berbeda per saham)
      </div>
    </div>
  );
}
