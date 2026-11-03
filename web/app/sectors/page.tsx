import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { Nav } from "@/components/layout/nav";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

export const instant = false;

// ── Types ──────────────────────────────────────────────────────────────────────

interface SectorData {
  ticker: string;
  name: string;
  price: number;
  change_pct: number;
}

interface SectorInsight {
  summary: string;
  leaders: string[];
  laggards: string[];
  watch: string[];
  sentiment: "positif" | "netral" | "negatif";
}

interface SectorsResponse {
  sectors: SectorData[];
  insight: SectorInsight | null;
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function changeColor(pct: number) {
  if (pct >= 1) return "text-green-600 dark:text-green-400";
  if (pct >= 0) return "text-green-500 dark:text-green-500";
  if (pct >= -1) return "text-amber-500 dark:text-amber-400";
  return "text-red-500 dark:text-red-400";
}

function changeEmoji(pct: number) {
  if (pct >= 1) return "🟢";
  if (pct >= 0) return "🟡";
  if (pct >= -1) return "🟠";
  return "🔴";
}

function sentimentStyle(s: string) {
  if (s === "positif") return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
  if (s === "negatif") return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
  return "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400";
}

function fmtChange(pct: number) {
  const sign = pct >= 0 ? "+" : "";
  return `${sign}${pct.toFixed(2)}%`;
}

// ── Diverging bar ──────────────────────────────────────────────────────────────

function SectorBar({ change, maxAbs }: { change: number; maxAbs: number }) {
  const pct = maxAbs === 0 ? 0 : Math.min(100, (Math.abs(change) / maxAbs) * 100);
  return (
    <div className="flex items-center gap-0.5 w-28 shrink-0">
      {/* Negative side */}
      <div className="w-13 flex justify-end overflow-hidden">
        {change < 0 && (
          <div
            className="h-1.5 rounded-l-full bg-red-400 dark:bg-red-500"
            style={{ width: `${pct}%` }}
          />
        )}
      </div>
      {/* Zero line */}
      <div className="w-px h-3 bg-zinc-300 dark:bg-zinc-600 flex-shrink-0" />
      {/* Positive side */}
      <div className="w-13 overflow-hidden">
        {change >= 0 && (
          <div
            className="h-1.5 rounded-r-full bg-green-400 dark:bg-green-500"
            style={{ width: `${pct}%` }}
          />
        )}
      </div>
    </div>
  );
}

// ── Main ───────────────────────────────────────────────────────────────────────

export default async function SectorsPage() {
  const session = await auth();
  const email = session?.user?.email ?? "";

  let data: SectorsResponse | null = null;
  let error = "";
  try {
    const res = await goFetch("/v1/sectors", email);
    if (res.ok) data = await res.json();
    else {
      const body = await res.json().catch(() => null);
      error = body?.error ?? `Error ${res.status}`;
    }
  } catch {
    error = "Gagal mengambil data sektor — pastikan Go API berjalan";
  }

  const sectors = data?.sectors ?? [];
  const insight = data?.insight ?? null;
  const maxAbs = sectors.length > 0 ? Math.max(...sectors.map((s) => Math.abs(s.change_pct))) : 1;

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
        <div className="max-w-2xl mx-auto space-y-6">
          {/* Header */}
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-50">Sektor US</h1>
            <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
              Rotasi sektor S&P 500 · update hari ini
            </p>
          </div>

          {error && <p className="text-sm text-red-500">{error}</p>}

          {/* AI Insight */}
          {insight && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                    AI Ringkasan
                  </p>
                  <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${sentimentStyle(insight.sentiment)}`}>
                    {insight.sentiment}
                  </span>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <p className="text-sm text-zinc-700 dark:text-zinc-300 leading-relaxed">
                    {insight.summary}
                  </p>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {insight.leaders.length > 0 && (
                      <div>
                        <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1.5 flex items-center gap-1">
                          <span>↑</span> Memimpin
                        </p>
                        <ul className="space-y-1">
                          {insight.leaders.map((l, i) => (
                            <li key={i} className="text-xs text-zinc-600 dark:text-zinc-400 flex gap-1.5">
                              <span className="text-green-500 flex-shrink-0 mt-0.5">•</span>
                              <span>{l}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {insight.laggards.length > 0 && (
                      <div>
                        <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1.5 flex items-center gap-1">
                          <span>↓</span> Tertinggal
                        </p>
                        <ul className="space-y-1">
                          {insight.laggards.map((l, i) => (
                            <li key={i} className="text-xs text-zinc-600 dark:text-zinc-400 flex gap-1.5">
                              <span className="text-red-400 flex-shrink-0 mt-0.5">•</span>
                              <span>{l}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>

                  {insight.watch.length > 0 && (
                    <div className="border-t border-zinc-100 dark:border-zinc-800 pt-3">
                      <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1.5">
                        👁 Perlu Dicermati untuk Riset Ticker
                      </p>
                      <ul className="space-y-1">
                        {insight.watch.map((w, i) => (
                          <li key={i} className="text-xs text-zinc-600 dark:text-zinc-400 flex gap-1.5">
                            <span className="text-blue-400 flex-shrink-0 mt-0.5">•</span>
                            <span>{w}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Sector list */}
          {sectors.length > 0 && (
            <Card>
              <CardHeader>
                <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                  Performa Hari Ini
                </p>
              </CardHeader>
              <CardContent className="px-0 pb-0">
                <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
                  {sectors.map((s) => (
                    <div
                      key={s.ticker}
                      className="flex items-center gap-3 px-4 py-2.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/30 transition-colors"
                    >
                      <span className="text-sm w-4 flex-shrink-0">{changeEmoji(s.change_pct)}</span>

                      <div className="w-10 flex-shrink-0">
                        <span className="text-xs font-bold text-zinc-800 dark:text-zinc-200">
                          {s.ticker}
                        </span>
                      </div>

                      <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-1 min-w-0 truncate">
                        {s.name}
                      </span>

                      <SectorBar change={s.change_pct} maxAbs={maxAbs} />

                      <span className={`text-xs font-semibold tabular-nums w-14 text-right flex-shrink-0 ${changeColor(s.change_pct)}`}>
                        {fmtChange(s.change_pct)}
                      </span>
                    </div>
                  ))}
                </div>

                <div className="px-4 py-2.5 border-t border-zinc-100 dark:border-zinc-800 text-[10px] text-zinc-400 dark:text-zinc-600">
                  Data: FMP · Perubahan harian dibanding penutupan sebelumnya
                </div>
              </CardContent>
            </Card>
          )}

          {!error && sectors.length === 0 && (
            <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-12">
              Data sektor belum tersedia. Pastikan FMP_API_KEY sudah dikonfigurasi.
            </p>
          )}
        </div>
      </main>
    </>
  );
}
