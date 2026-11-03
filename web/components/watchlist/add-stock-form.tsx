"use client";

import { useActionState } from "react";
import { Button } from "@/components/ui/button";
import { addToWatchlist } from "@/app/watchlist/actions";

interface FormState {
  error?: string;
}

export function AddStockForm() {
  const [state, formAction, pending] = useActionState<FormState, FormData>(
    async (_prev, formData) => {
      const ticker = (formData.get("ticker") as string)?.trim().toUpperCase();
      const name = (formData.get("name") as string)?.trim();
      const status = (formData.get("status") as string) || "watching";

      if (!ticker) return { error: "Ticker wajib diisi" };

      try {
        await addToWatchlist(ticker, name || ticker, status);
        return {};
      } catch {
        return { error: "Gagal menambahkan saham" };
      }
    },
    {}
  );

  return (
    <form action={formAction} className="flex flex-col gap-2">
      {/* Row 1: ticker + status (always side by side) */}
      <div className="flex gap-2">
        <input
          name="ticker"
          placeholder="Ticker (contoh: AAPL)"
          required
          className="flex-1 min-w-0 px-3 py-2 text-sm rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-50 placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-400"
        />
        <select
          name="status"
          defaultValue="watching"
          className="px-3 py-2 text-sm rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-50 focus:outline-none focus:ring-2 focus:ring-zinc-400 shrink-0"
        >
          <option value="watching">Watching</option>
          <option value="holding">Holding</option>
        </select>
      </div>

      {/* Row 2: name + submit */}
      <div className="flex gap-2">
        <input
          name="name"
          placeholder="Nama perusahaan (opsional)"
          className="flex-1 min-w-0 px-3 py-2 text-sm rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-50 placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-400"
        />
        <Button type="submit" loading={pending} className="shrink-0">
          Tambah
        </Button>
      </div>

      {state.error && (
        <p className="text-xs text-red-500">{state.error}</p>
      )}
    </form>
  );
}
