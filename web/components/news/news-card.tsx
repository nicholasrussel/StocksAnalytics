import { Card } from "@/components/ui/card";

export interface NewsItem {
  headline: string;
  source: string;
  url: string;
  summary: string;
  image: string;
  category: string;
  published_at: string;
}

interface NewsCardProps {
  item: NewsItem;
}

export function NewsCard({ item }: NewsCardProps) {
  const date = new Date(item.published_at).toLocaleDateString("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <Card className="flex flex-col gap-2 p-4 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {item.source}
        </span>
        <span className="text-xs text-zinc-400 dark:text-zinc-500 shrink-0">
          {date}
        </span>
      </div>

      <a
        href={item.url}
        target="_blank"
        rel="noopener noreferrer"
        className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 hover:underline leading-snug"
      >
        {item.headline}
      </a>

      {item.summary && (
        <p className="text-xs text-zinc-500 dark:text-zinc-400 line-clamp-2 leading-relaxed">
          {item.summary}
        </p>
      )}
    </Card>
  );
}
