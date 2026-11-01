"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { removeFromWatchlist } from "@/app/watchlist/actions";

export interface WatchlistItemData {
  ticker: string;
  name: string;
  status: string;
  note?: string;
  added_at: string;
}

export function WatchlistItem({ item }: { item: WatchlistItemData }) {
  return (
    <div className="flex items-center justify-between px-4 py-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors">
      <Link
        href={`/watchlist/${item.ticker}`}
        className="flex items-center gap-3 min-w-0 flex-1"
      >
        <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-zinc-100 dark:bg-zinc-800 shrink-0">
          <span className="text-xs font-bold text-zinc-700 dark:text-zinc-300">
            {item.ticker.slice(0, 4)}
          </span>
        </div>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 truncate">
            {item.ticker}
          </p>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
            {item.name}
          </p>
        </div>
      </Link>

      <div className="flex items-center gap-2 shrink-0 ml-2">
        <span
          className={[
            "text-xs px-2 py-0.5 rounded-full font-medium",
            item.status === "holding"
              ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
              : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
          ].join(" ")}
        >
          {item.status}
        </span>

        <form action={removeFromWatchlist.bind(null, item.ticker)}>
          <Button
            type="submit"
            variant="ghost"
            size="sm"
            className="text-zinc-400 hover:text-red-500"
          >
            ×
          </Button>
        </form>
      </div>
    </div>
  );
}
