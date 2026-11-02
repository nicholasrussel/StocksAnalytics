import { goFetch } from "@/lib/go-api";
import { Tooltip } from "@/components/ui/tooltip";
import { getDefinition, type MetricKey } from "@/lib/metric-definitions";

// ── Types ──────────────────────────────────────────────────────────────────────

interface Scorecard {
  ticker: string;
  period_end: string;
  period: string;
  // Profitabilitas
  revenue: number;
  revenue_growth?: number;
  gross_margin?: number;
  operating_margin?: number;
  ebitda_margin?: number;
  net_margin?: number;
  eps: number;
  eps_growth?: number;
  roe?: number;
  roa?: number;
  roic?: number;
  // Likuiditas
  current_ratio?: number;
  quick_ratio?: number;
  cash_position: number;
  // Leverage
  debt_to_equity?: number;
  net_debt: number;
  net_debt_to_ebitda?: number;
  interest_coverage?: number;
  // Arus Kas
  operating_cash_flow: number;
  capex: number;
  free_cash_flow: number;
  fcf_margin?: number;
  fcf_conversion?: number;
  // Valuasi
  price?: number;
  market_cap?: number;
  pe_ratio?: number;
  ps_ratio?: number;
  pb_ratio?: number;
  ev_ebitda?: number;
  // Skor
  piotroski_score: number;
  piotroski_grade: string;
  flags: Flag[];
}

interface Flag {
  level: "ok" | "warning" | "danger";
  metric: string;
  message: string;
}

interface Insight {
  summary: string;
  strengths: string[];
  risks: string[];
  outlook: "positif" | "netral" | "negatif";
  context_note: string;
}

interface FundamentalsResponse {
  scorecard: Scorecard;
  insight: Insight | null;
  cached: boolean;
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function pct(v?: number, decimals = 1) {
  if (v == null) return "—";
  return (v * 100).toFixed(decimals) + "%";
}

function num(v?: number, decimals = 2) {
  if (v == null) return "—";
  if (Math.abs(v) >= 1e12) return (v / 1e12).toFixed(1) + "T";
  if (Math.abs(v) >= 1e9) return (v / 1e9).toFixed(1) + "B";
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(1) + "M";
  return v.toFixed(decimals);
}

function ratio(v?: number) {
  if (v == null) return "—";
  return v.toFixed(2) + "x";
}

function flagColor(level: Flag["level"]) {
  if (level === "ok") return "text-green-600 dark:text-green-400";
  if (level === "danger") return "text-red-500 dark:text-red-400";
  return "text-amber-500 dark:text-amber-400";
}
function flagDot(level: Flag["level"]) {
  if (level === "ok") return "🟢";
  if (level === "danger") return "🔴";
  return "🟡";
}
function outlookColor(o: string) {
  if (o === "positif") return "text-green-600 dark:text-green-400";
  if (o === "negatif") return "text-red-500 dark:text-red-400";
  return "text-amber-500 dark:text-amber-400";
}

function piotroskiColor(grade: string) {
  if (grade === "Sehat") return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
  if (grade === "Lemah") return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
  return "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400";
}

function growth(v?: number) {
  if (v == null) return null;
  const sign = v >= 0 ? "+" : "";
  const color = v >= 0 ? "text-green-600 dark:text-green-400" : "text-red-500 dark:text-red-400";
  return <span className={`text-xs font-medium ${color}`}>{sign}{(v * 100).toFixed(1)}% YoY</span>;
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function TooltipContent({ metricKey }: { metricKey: MetricKey }) {
  const def = getDefinition(metricKey);
  if (!def) return null;
  return (
    <span className="space-y-1.5 block">
      <span className="font-semibold text-zinc-800 dark:text-zinc-100 block">{def.label}</span>
      <span className="block">{def.definition}</span>
      <span className="block text-zinc-400 dark:text-zinc-500 border-t border-zinc-100 dark:border-zinc-800 pt-1.5 mt-1.5">
        <span className="font-medium text-zinc-500 dark:text-zinc-400">Interpretasi: </span>
        {def.interpretation}
      </span>
    </span>
  );
}

function MetricRow({
  label,
  metricKey,
  value,
  sub,
}: {
  label: string;
  metricKey?: MetricKey;
  value: React.ReactNode;
  sub?: React.ReactNode;
}) {
  return (
    <div className="flex items-baseline justify-between py-1.5 border-b border-zinc-100 dark:border-zinc-800 last:border-0">
      <span className="text-xs text-zinc-500 dark:text-zinc-400">
        {metricKey ? (
          <Tooltip content={<TooltipContent metricKey={metricKey} />}>{label}</Tooltip>
        ) : label}
      </span>
      <span className="text-sm font-medium text-zinc-800 dark:text-zinc-200 flex items-center gap-2">
        {value}
        {sub}
      </span>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-[10px] font-semibold text-zinc-400 dark:text-zinc-600 uppercase tracking-widest mb-2">{title}</p>
      {children}
    </div>
  );
}

// ── Skeleton ───────────────────────────────────────────────────────────────────

export function FundamentalsAnalysisSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      {[...Array(4)].map((_, i) => (
        <div key={i} className="space-y-2">
          <div className="h-3 w-24 bg-zinc-200 dark:bg-zinc-700 rounded" />
          {[...Array(3)].map((_, j) => (
            <div key={j} className="h-5 w-full bg-zinc-100 dark:bg-zinc-800 rounded" />
          ))}
        </div>
      ))}
    </div>
  );
}

// ── Main component (async server component) ────────────────────────────────────

export async function FundamentalsAnalysis({ ticker, email }: { ticker: string; email: string }) {
  let data: FundamentalsResponse | null = null;
  let error = "";

  try {
    const res = await goFetch(`/v1/stocks/${ticker}/fundamentals`, email);
    if (res.ok) {
      data = await res.json();
    } else {
      const body = await res.json().catch(() => null);
      error = body?.error ?? `Error ${res.status}`;
    }
  } catch {
    error = "Gagal mengambil data laporan keuangan";
  }

  if (error || !data?.scorecard) {
    return (
      <p className="text-sm text-red-500 dark:text-red-400">
        {error || "Data laporan keuangan tidak tersedia. Pastikan FMP_API_KEY sudah dikonfigurasi."}
      </p>
    );
  }

  const { scorecard: sc, insight } = data;

  return (
    <div className="space-y-5">
      {/* Header: period + Piotroski */}
      <div className="flex items-center justify-between">
        <p className="text-xs text-zinc-400 dark:text-zinc-500">
          {sc.period} {sc.period_end?.slice(0, 4)}
          {data.cached && <span className="ml-2 text-zinc-300 dark:text-zinc-600">(dari cache)</span>}
        </p>
        <Tooltip content={<TooltipContent metricKey="piotroski" />}>
          <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${piotroskiColor(sc.piotroski_grade)}`}>
            Piotroski {sc.piotroski_score}/9 · {sc.piotroski_grade}
          </span>
        </Tooltip>
      </div>

      {/* Profitabilitas */}
      <Section title="Profitabilitas">
        <MetricRow metricKey="revenue" label="Pendapatan" value={`$${num(sc.revenue)}`} sub={growth(sc.revenue_growth)} />
        <MetricRow metricKey="gross_margin" label="Gross Margin" value={pct(sc.gross_margin)} />
        <MetricRow metricKey="operating_margin" label="Operating Margin" value={pct(sc.operating_margin)} />
        <MetricRow metricKey="ebitda_margin" label="EBITDA Margin" value={pct(sc.ebitda_margin)} />
        <MetricRow metricKey="net_margin" label="Net Margin" value={pct(sc.net_margin)} />
        <MetricRow metricKey="eps" label="EPS (diluted)" value={`$${num(sc.eps)}`} sub={growth(sc.eps_growth)} />
        <MetricRow metricKey="roe" label="ROE" value={pct(sc.roe)} />
        <MetricRow metricKey="roa" label="ROA" value={pct(sc.roa)} />
        {sc.roic != null && <MetricRow metricKey="roic" label="ROIC" value={pct(sc.roic)} />}
      </Section>

      {/* Likuiditas */}
      <Section title="Likuiditas">
        <MetricRow metricKey="current_ratio" label="Current Ratio" value={ratio(sc.current_ratio)} />
        <MetricRow metricKey="quick_ratio" label="Quick Ratio" value={ratio(sc.quick_ratio)} />
        <MetricRow metricKey="cash_position" label="Kas & Setara" value={`$${num(sc.cash_position)}`} />
      </Section>

      {/* Leverage */}
      <Section title="Solvabilitas & Leverage">
        <MetricRow metricKey="debt_to_equity" label="Debt / Equity" value={ratio(sc.debt_to_equity)} />
        <MetricRow
          metricKey="net_debt"
          label="Net Debt"
          value={sc.net_debt < 0
            ? <span className="text-green-600 dark:text-green-400">Net Cash ${num(Math.abs(sc.net_debt))}</span>
            : `$${num(sc.net_debt)}`}
        />
        <MetricRow metricKey="net_debt_to_ebitda" label="Net Debt / EBITDA" value={ratio(sc.net_debt_to_ebitda)} />
        <MetricRow metricKey="interest_coverage" label="Interest Coverage" value={ratio(sc.interest_coverage)} />
      </Section>

      {/* Arus Kas */}
      <Section title="Arus Kas">
        <MetricRow metricKey="operating_cash_flow" label="Operating Cash Flow" value={`$${num(sc.operating_cash_flow)}`} />
        <MetricRow metricKey="capex" label="CapEx" value={`$${num(Math.abs(sc.capex))}`} />
        <MetricRow metricKey="free_cash_flow" label="Free Cash Flow" value={`$${num(sc.free_cash_flow)}`} />
        <MetricRow metricKey="fcf_margin" label="FCF Margin" value={pct(sc.fcf_margin)} />
        {sc.fcf_conversion != null && (
          <MetricRow metricKey="fcf_conversion" label="FCF Conversion" value={pct(sc.fcf_conversion, 0)} />
        )}
      </Section>

      {/* Valuasi */}
      {(sc.pe_ratio ?? sc.ps_ratio ?? sc.pb_ratio ?? sc.ev_ebitda) != null && (
        <Section title="Valuasi">
          {sc.price != null && <MetricRow label="Harga Saham" value={`$${sc.price.toFixed(2)}`} />}
          {sc.pe_ratio != null && <MetricRow metricKey="pe_ratio" label="P/E" value={ratio(sc.pe_ratio)} />}
          {sc.ps_ratio != null && <MetricRow metricKey="ps_ratio" label="P/S" value={ratio(sc.ps_ratio)} />}
          {sc.pb_ratio != null && <MetricRow metricKey="pb_ratio" label="P/B" value={ratio(sc.pb_ratio)} />}
          {sc.ev_ebitda != null && <MetricRow metricKey="ev_ebitda" label="EV / EBITDA" value={ratio(sc.ev_ebitda)} />}
        </Section>
      )}

      {/* Flags */}
      {sc.flags?.length > 0 && (
        <Section title="Sinyal Penting">
          <div className="space-y-1.5">
            {sc.flags.map((f, i) => (
              <div key={i} className={`text-xs flex gap-2 ${flagColor(f.level)}`}>
                <span>{flagDot(f.level)}</span>
                <span>{f.message}</span>
              </div>
            ))}
          </div>
        </Section>
      )}

      {/* LLM Insight */}
      {insight && (
        <Section title="AI Insight">
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-500 dark:text-zinc-400">Outlook:</span>
              <span className={`text-xs font-semibold capitalize ${outlookColor(insight.outlook)}`}>
                {insight.outlook}
              </span>
            </div>
            <p className="text-sm text-zinc-700 dark:text-zinc-300 leading-relaxed">{insight.summary}</p>
            {insight.strengths?.length > 0 && (
              <div>
                <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Kekuatan:</p>
                <ul className="space-y-1">
                  {insight.strengths.map((s, i) => (
                    <li key={i} className="text-xs text-zinc-600 dark:text-zinc-400 flex gap-1.5">
                      <span className="text-green-500 flex-shrink-0">✓</span>{s}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {insight.risks?.length > 0 && (
              <div>
                <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Risiko:</p>
                <ul className="space-y-1">
                  {insight.risks.map((r, i) => (
                    <li key={i} className="text-xs text-zinc-600 dark:text-zinc-400 flex gap-1.5">
                      <span className="text-amber-500 flex-shrink-0">⚠</span>{r}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {insight.context_note && (
              <p className="text-xs text-zinc-500 dark:text-zinc-500 italic border-l-2 border-zinc-200 dark:border-zinc-700 pl-3">
                {insight.context_note}
              </p>
            )}
          </div>
        </Section>
      )}
    </div>
  );
}
