import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { NewsCard, type NewsItem } from "@/components/news/news-card";
import { Nav } from "@/components/layout/nav";

export const instant = false;

interface MarketsData {
  global: NewsItem[];
  btc: NewsItem[];
}

export default async function MarketsPage() {
  const session = await auth();
  const email = session?.user?.email ?? "";

  let data: MarketsData = { global: [], btc: [] };
  let error = "";
  try {
    const res = await goFetch("/v1/markets/news", email);
    if (res.ok) data = await res.json();
    else error = `Error ${res.status}`;
  } catch {
    error = "Gagal mengambil berita — pastikan Go API berjalan";
  }

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
        <div className="max-w-2xl mx-auto space-y-10">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-50">Markets</h1>
            <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
              Ekonomi global, minyak, geopolitik &amp; BTC
            </p>
          </div>

          {error && <p className="text-sm text-red-500">{error}</p>}

          {/* Global Markets */}
          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">
                Ekonomi Global
              </h2>
              <span className="text-xs text-zinc-400">({data.global.length})</span>
            </div>

            {data.global.length === 0 && !error ? (
              <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-6">
                Tidak ada berita tersedia.
              </p>
            ) : (
              <div className="space-y-3">
                {data.global.map((item, i) => (
                  <NewsCard key={i} item={item} />
                ))}
              </div>
            )}
          </section>

          {/* BTC */}
          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-zinc-700 dark:text-zinc-300">
                Bitcoin (BTC)
              </h2>
              <span className="text-xs text-zinc-400">({data.btc.length})</span>
            </div>

            {data.btc.length === 0 && !error ? (
              <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-6">
                Tidak ada berita BTC tersedia.
              </p>
            ) : (
              <div className="space-y-3">
                {data.btc.map((item, i) => (
                  <NewsCard key={i} item={item} />
                ))}
              </div>
            )}
          </section>
        </div>
      </main>
    </>
  );
}
