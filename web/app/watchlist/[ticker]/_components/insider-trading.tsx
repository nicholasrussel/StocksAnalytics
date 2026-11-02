import { goFetch } from "@/lib/go-api";

interface InsiderTransaction {
  reporter_name: string;
  officer_title: string;
  is_director: boolean;
  is_officer: boolean;
  transaction_date: string;
  filing_date: string;
  transaction_code: string; // P=buy, S=sell, A=award, F=tax, M=option, G=gift
  shares: number;
  price_per_share: number;
  shares_owned: number;
  acq_disp: string; // A or D
  security_title: string;
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function shortName(full: string) {
  // EDGAR names come as "COOK TIMOTHY D" — convert to "Timothy Cook"
  const parts = full.trim().split(/\s+/);
  if (parts.length === 1) return full;
  // Capitalize each part, take first + last
  const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
  return cap(parts[parts.length - 1]) + " " + cap(parts[0]);
}

function shortTitle(title: string, isDirector: boolean, isOfficer: boolean) {
  if (title) {
    const t = title.toLowerCase();
    if (t.includes("chief executive") || t.includes("ceo")) return "CEO";
    if (t.includes("chief financial") || t.includes("cfo")) return "CFO";
    if (t.includes("chief operating") || t.includes("coo")) return "COO";
    if (t.includes("chief technology") || t.includes("cto")) return "CTO";
    if (t.includes("president")) return "President";
    if (t.includes("chairman")) return "Chairman";
    if (t.includes("general counsel")) return "General Counsel";
    return title.slice(0, 22);
  }
  if (isDirector) return "Director";
  if (isOfficer) return "Officer";
  return "Insider";
}

type TxType = "buy" | "sell" | "award" | "other";

function classifyTx(code: string): TxType {
  switch (code.toUpperCase()) {
    case "P": return "buy";
    case "S": return "sell";
    case "A": return "award";
    default:  return "other";
  }
}

function txLabel(type: TxType) {
  switch (type) {
    case "buy": return "BELI";
    case "sell": return "JUAL";
    case "award": return "AWARD";
    default: return "LAINNYA";
  }
}

function txBadge(type: TxType) {
  switch (type) {
    case "buy":
      return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
    case "sell":
      return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
    case "award":
      return "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400";
    default:
      return "bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400";
  }
}

function fmt(v: number) {
  if (Math.abs(v) >= 1e9) return (v / 1e9).toFixed(1) + "B";
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(1) + "M";
  if (Math.abs(v) >= 1e3) return (v / 1e3).toFixed(0) + "K";
  return v.toLocaleString("en-US", { maximumFractionDigits: 0 });
}

function relDate(dateStr: string) {
  const d = new Date(dateStr);
  const diff = Math.floor((Date.now() - d.getTime()) / (1000 * 60 * 60 * 24));
  if (diff === 0) return "hari ini";
  if (diff === 1) return "kemarin";
  if (diff < 30) return `${diff}h lalu`;
  if (diff < 365) return `${Math.floor(diff / 30)}bln lalu`;
  return `${Math.floor(diff / 365)}thn lalu`;
}

// ── Skeleton ───────────────────────────────────────────────────────────────────

export function InsiderTradingSkeleton() {
  return (
    <div className="space-y-3 animate-pulse">
      <div className="h-3 w-40 bg-zinc-200 dark:bg-zinc-700 rounded" />
      {[...Array(4)].map((_, i) => (
        <div key={i} className="flex items-center gap-3">
          <div className="h-5 w-12 bg-zinc-100 dark:bg-zinc-800 rounded-full" />
          <div className="flex-1 h-4 bg-zinc-100 dark:bg-zinc-800 rounded" />
          <div className="h-4 w-16 bg-zinc-100 dark:bg-zinc-800 rounded" />
        </div>
      ))}
    </div>
  );
}

// ── Main ───────────────────────────────────────────────────────────────────────

export async function InsiderTrading({ ticker, email }: { ticker: string; email: string }) {
  let txns: InsiderTransaction[] = [];
  let error = "";

  try {
    const res = await goFetch(`/v1/stocks/${ticker}/insider-trading`, email);
    if (res.ok) {
      txns = await res.json();
    } else {
      const body = await res.json().catch(() => null);
      error = body?.error ?? `Error ${res.status}`;
    }
  } catch {
    error = "Gagal mengambil data insider trading";
  }

  if (error) {
    return <p className="text-sm text-red-500 dark:text-red-400">{error}</p>;
  }

  if (!txns.length) {
    return (
      <p className="text-sm text-zinc-400 dark:text-zinc-600 text-center py-4">
        Tidak ada data transaksi insider untuk {ticker}.
      </p>
    );
  }

  const buys = txns.filter(t => classifyTx(t.transaction_code) === "buy").length;
  const sells = txns.filter(t => classifyTx(t.transaction_code) === "sell").length;

  return (
    <div className="space-y-3">
      {/* Summary */}
      <div className="flex items-center gap-3 text-xs text-zinc-400 dark:text-zinc-500">
        {buys > 0 && (
          <span className="flex items-center gap-1">
            <span className="text-green-500 font-semibold">{buys}</span> pembelian
          </span>
        )}
        {sells > 0 && (
          <span className="flex items-center gap-1">
            <span className="text-red-500 font-semibold">{sells}</span> penjualan
          </span>
        )}
        <span className="ml-auto">15 transaksi terbaru</span>
      </div>

      {/* Transaction list */}
      <div className="space-y-1.5">
        {txns.map((t, i) => {
          const type = classifyTx(t.transaction_code);
          const value = t.shares * t.price_per_share;
          return (
            <div
              key={i}
              className="flex items-center gap-2.5 py-1.5 border-b border-zinc-100 dark:border-zinc-800 last:border-0"
            >
              {/* Type badge */}
              <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded flex-shrink-0 ${txBadge(type)}`}>
                {txLabel(type)}
              </span>

              {/* Name + title */}
              <div className="flex-1 min-w-0">
                <span className="text-xs font-medium text-zinc-700 dark:text-zinc-300 truncate block">
                  {shortName(t.reporter_name)}
                </span>
                <span className="text-[10px] text-zinc-400 dark:text-zinc-500">
                  {shortTitle(t.officer_title, t.is_director, t.is_officer)}
                </span>
              </div>

              {/* Shares + value */}
              <div className="text-right flex-shrink-0">
                <span className="text-xs font-medium text-zinc-700 dark:text-zinc-200 block">
                  {fmt(t.shares)} lbr
                </span>
                {value > 0 && (
                  <span className="text-[10px] text-zinc-400 dark:text-zinc-500">
                    ~${fmt(value)}
                  </span>
                )}
              </div>

              {/* Date */}
              <span className="text-[10px] text-zinc-400 dark:text-zinc-500 flex-shrink-0 w-16 text-right">
                {relDate(t.transaction_date || t.filing_date)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
