"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { WatchlistItem, type WatchlistItemData } from "./watchlist-item";
import { Button } from "@/components/ui/button";

export function CompareSelector({ items }: { items: WatchlistItemData[] }) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const router = useRouter();

  const toggle = (ticker: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(ticker)) {
        next.delete(ticker);
      } else if (next.size < 5) {
        next.add(ticker);
      }
      return next;
    });
  };

  const handleCompare = () => {
    router.push(`/watchlist/compare?tickers=${[...selected].join(",")}`);
  };

  return (
    <>
      <div className="space-y-2">
        {items.length === 0 ? (
          <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-10">
            Belum ada saham. Tambahkan di atas.
          </p>
        ) : (
          items.map((item) => (
            <WatchlistItem
              key={item.ticker}
              item={item}
              selected={selected.has(item.ticker)}
              onToggle={() => toggle(item.ticker)}
            />
          ))
        )}
      </div>

      {/* Floating compare bar */}
      {selected.size >= 2 && (
        <div className="fixed bottom-20 md:bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 px-5 py-3 rounded-full shadow-xl border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900">
          <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
            {selected.size} saham dipilih
          </span>
          <div className="flex items-center gap-1">
            {[...selected].map((t) => (
              <span
                key={t}
                className="text-xs font-bold px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300"
              >
                {t}
              </span>
            ))}
          </div>
          <Button size="sm" onClick={handleCompare}>
            Bandingkan →
          </Button>
          <button
            onClick={() => setSelected(new Set())}
            className="text-xs text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
          >
            Batal
          </button>
        </div>
      )}
    </>
  );
}
