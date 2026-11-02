import { Suspense } from "react";
import Link from "next/link";
import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { NewsCard, type NewsItem } from "@/components/news/news-card";
import { Button } from "@/components/ui/button";
import { Nav } from "@/components/layout/nav";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { StockAnalysis, StockAnalysisSkeleton } from "./_components/stock-analysis";
import { FundamentalsAnalysis, FundamentalsAnalysisSkeleton } from "./_components/fundamentals";
import { InsiderTrading, InsiderTradingSkeleton } from "./_components/insider-trading";

export const instant = false;

interface Props {
  params: Promise<{ ticker: string }>;
}

export default async function StockPage({ params }: Props) {
  const { ticker } = await params;
  const session = await auth();
  const email = session?.user?.email ?? "";
  const symbol = ticker.toUpperCase();

  let news: NewsItem[] = [];
  let newsError = "";
  try {
    const res = await goFetch(`/v1/stocks/${symbol}/news`, email);
    if (res.ok) news = await res.json();
    else newsError = `Error ${res.status}`;
  } catch {
    newsError = "Gagal mengambil berita — pastikan Go API berjalan";
  }

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
        <div className="max-w-2xl mx-auto space-y-6">
          {/* Header */}
          <div className="flex items-center gap-3">
            <Link href="/watchlist">
              <Button variant="ghost" size="sm">← Kembali</Button>
            </Link>
            <div>
              <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-50">{symbol}</h1>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">Berita 7 hari terakhir</p>
            </div>
          </div>

          {/* Insider Trading (Form 4) */}
          <Card>
            <CardHeader>
              <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                Transaksi Insider
              </p>
            </CardHeader>
            <CardContent>
              <Suspense fallback={<InsiderTradingSkeleton />}>
                <InsiderTrading ticker={symbol} email={email} />
              </Suspense>
            </CardContent>
          </Card>

          {/* Laporan Keuangan + AI Insight */}
          <Card>
            <CardHeader>
              <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                Laporan Keuangan
              </p>
            </CardHeader>
            <CardContent>
              <Suspense fallback={<FundamentalsAnalysisSkeleton />}>
                <FundamentalsAnalysis ticker={symbol} email={email} />
              </Suspense>
            </CardContent>
          </Card>

          {/* LLM Analysis (news-based) */}
          <Card>
            <CardHeader>
              <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                Ringkasan Berita AI
              </p>
            </CardHeader>
            <CardContent>
              <Suspense fallback={<StockAnalysisSkeleton />}>
                <StockAnalysis ticker={symbol} email={email} />
              </Suspense>
            </CardContent>
          </Card>

          {/* News list */}
          <div className="space-y-3">
            <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">
              Berita ({news.length})
            </p>

            {newsError ? (
              <p className="text-sm text-red-500">{newsError}</p>
            ) : news.length === 0 ? (
              <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-8">
                Tidak ada berita ditemukan untuk {symbol}.
              </p>
            ) : (
              news.map((item, i) => <NewsCard key={i} item={item} />)
            )}
          </div>
        </div>
      </main>
    </>
  );
}
