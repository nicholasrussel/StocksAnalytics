import Link from "next/link";
import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { Nav } from "@/components/layout/nav";
import { Button } from "@/components/ui/button";
import { ComparisonTable } from "./_components/comparison-table";

export const instant = false;

interface Props {
  searchParams: Promise<{ tickers?: string }>;
}

export default async function ComparePage({ searchParams }: Props) {
  const { tickers: tickersParam } = await searchParams;
  const session = await auth();
  const email = session?.user?.email ?? "";

  const tickers = (tickersParam ?? "")
    .split(",")
    .map((t) => t.trim().toUpperCase())
    .filter(Boolean)
    .slice(0, 5);

  if (tickers.length < 2) {
    return (
      <>
        <Nav />
        <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
          <div className="max-w-4xl mx-auto space-y-4">
            <Link href="/watchlist">
              <Button variant="ghost" size="sm">← Kembali</Button>
            </Link>
            <p className="text-sm text-zinc-500">
              Pilih minimal 2 saham dari halaman watchlist untuk dibandingkan.
            </p>
          </div>
        </main>
      </>
    );
  }

  let entries: CompareEntry[] = [];
  let fetchError = "";
  try {
    const res = await goFetch(`/v1/stocks/compare?tickers=${tickers.join(",")}`, email);
    if (res.ok) entries = await res.json();
    else fetchError = `Error ${res.status}`;
  } catch {
    fetchError = "Gagal mengambil data perbandingan";
  }

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
        <div className="max-w-5xl mx-auto space-y-6">
          {/* Header */}
          <div className="flex items-center gap-3">
            <Link href="/watchlist">
              <Button variant="ghost" size="sm">← Watchlist</Button>
            </Link>
            <div>
              <h1 className="text-xl font-bold text-zinc-900 dark:text-zinc-50">
                Perbandingan Saham
              </h1>
              <p className="text-xs text-zinc-500 dark:text-zinc-400">
                {tickers.join(" · ")}
              </p>
            </div>
          </div>

          {fetchError ? (
            <p className="text-sm text-red-500">{fetchError}</p>
          ) : (
            <ComparisonTable entries={entries} />
          )}
        </div>
      </main>
    </>
  );
}

// Type exported for use in sub-components
export interface CompareEntry {
  ticker: string;
  period_end?: string;
  computed_at?: string;
  scorecard?: Scorecard;
}

export interface Scorecard {
  ticker: string;
  period_end: string;
  period: string;
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
  current_ratio?: number;
  quick_ratio?: number;
  cash_position: number;
  debt_to_equity?: number;
  net_debt: number;
  net_debt_to_ebitda?: number;
  interest_coverage?: number;
  operating_cash_flow: number;
  capex: number;
  free_cash_flow: number;
  fcf_margin?: number;
  fcf_conversion?: number;
  price?: number;
  market_cap?: number;
  pe_ratio?: number;
  ps_ratio?: number;
  pb_ratio?: number;
  ev_ebitda?: number;
  piotroski_score: number;
  piotroski_grade: string;
}
