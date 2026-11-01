import Link from "next/link";
import { auth, signOut } from "@/auth";

export async function Nav() {
  const session = await auth();

  return (
    <header className="sticky top-0 z-10 bg-white/80 dark:bg-zinc-950/80 backdrop-blur border-b border-zinc-200 dark:border-zinc-800">
      <div className="max-w-2xl mx-auto px-4 h-12 flex items-center justify-between">
        <nav className="flex items-center gap-1">
          <Link
            href="/watchlist"
            className="px-3 py-1.5 text-sm font-medium rounded-md text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          >
            Watchlist
          </Link>
          <Link
            href="/markets"
            className="px-3 py-1.5 text-sm font-medium rounded-md text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          >
            Markets
          </Link>
          <Link
            href="/settings"
            className="px-3 py-1.5 text-sm font-medium rounded-md text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          >
            Pengaturan
          </Link>
        </nav>

        {session?.user?.email && (
          <form
            action={async () => {
              "use server";
              await signOut({ redirectTo: "/login" });
            }}
          >
            <button
              type="submit"
              className="text-xs text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors"
            >
              Keluar
            </button>
          </form>
        )}
      </div>
    </header>
  );
}
