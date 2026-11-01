import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { AddStockForm } from "@/components/watchlist/add-stock-form";
import { WatchlistItem, type WatchlistItemData } from "@/components/watchlist/watchlist-item";
import { Nav } from "@/components/layout/nav";

export const instant = false;

export default async function WatchlistPage() {
  const session = await auth();
  const email = session?.user?.email ?? "";

  let items: WatchlistItemData[] = [];
  try {
    const res = await goFetch("/v1/watchlist", email);
    if (res.ok) items = await res.json();
  } catch {
    // silently handle — show empty state
  }

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
      <div className="max-w-2xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-50">Watchlist</h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
            Saham yang kamu pantau dan koleksi
          </p>
        </div>

        <AddStockForm />

        <div className="space-y-2">
          {items.length === 0 ? (
            <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-10">
              Belum ada saham. Tambahkan di atas.
            </p>
          ) : (
            items.map((item) => <WatchlistItem key={item.ticker} item={item} />)
          )}
        </div>
      </div>
    </main>
    </>
  );
}
