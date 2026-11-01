import { goFetch } from "@/lib/go-api";

interface Analysis {
  sentiment: "positif" | "netral" | "negatif";
  summary: string;
  highlights: string[];
}

const sentimentStyle: Record<string, string> = {
  positif: "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20",
  negatif: "text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20",
  netral: "text-zinc-600 dark:text-zinc-400 bg-zinc-100 dark:bg-zinc-800",
};

export async function StockAnalysis({ ticker, email }: { ticker: string; email: string }) {
  let analysis: Analysis | null = null;
  let errorMsg = "";

  try {
    const res = await goFetch(`/v1/stocks/${ticker}/analyze`, email);
    if (res.ok) {
      analysis = await res.json();
      console.log("json analysis ==",analysis)
    } else {
      const body = await res.json().catch(() => null);
      errorMsg = body?.error ?? `HTTP ${res.status}`;
    }
  } catch (e) {
    errorMsg = e instanceof Error ? e.message : "Koneksi ke Go API gagal";
  }

  if (errorMsg) {
    return (
      <p className="text-sm text-red-500 font-mono">
        Error: {errorMsg}
      </p>
    );
  }

  if (!analysis) {
    return (
      <p className="text-sm text-zinc-400 dark:text-zinc-600 italic">
        Analisis tidak tersedia saat ini.
      </p>
    );
  }

  const style = sentimentStyle[analysis.sentiment] ?? sentimentStyle.netral;

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <span className={`text-xs font-semibold px-2 py-0.5 rounded-full capitalize ${style}`}>
          {analysis.sentiment}
        </span>
      </div>
      <p className="text-sm text-zinc-700 dark:text-zinc-300 leading-relaxed">
        {analysis.summary}
      </p>
      {analysis.highlights.length > 0 && (
        <ul className="space-y-1">
          {analysis.highlights.map((h, i) => (
            <li key={i} className="flex items-start gap-2 text-xs text-zinc-500 dark:text-zinc-400">
              <span className="shrink-0 mt-0.5">•</span>
              <span>{h}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function StockAnalysisSkeleton() {
  return (
    <div className="space-y-2 animate-pulse">
      <div className="h-5 w-16 rounded-full bg-zinc-200 dark:bg-zinc-700" />
      <div className="h-4 w-full rounded bg-zinc-200 dark:bg-zinc-700" />
      <div className="h-4 w-5/6 rounded bg-zinc-200 dark:bg-zinc-700" />
      <div className="h-4 w-4/6 rounded bg-zinc-200 dark:bg-zinc-700" />
    </div>
  );
}
