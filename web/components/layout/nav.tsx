import { auth, signOut } from "@/auth";
import { NavItem } from "./nav-item";

// ── Icons ──────────────────────────────────────────────────────────────────────

function ClipboardIcon() {
  return (
    <svg fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24" className="w-full h-full">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 011-1h2a2 2 0 011 1M9 12h6M9 15h4" />
    </svg>
  );
}

function GlobeIcon() {
  return (
    <svg fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24" className="w-full h-full">
      <circle cx="12" cy="12" r="9" strokeLinecap="round" strokeLinejoin="round" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M3.6 9h16.8M3.6 15h16.8M12 3c-2 2.5-3.2 5.6-3.2 9s1.2 6.5 3.2 9m0-18c2 2.5 3.2 5.6 3.2 9s-1.2 6.5-3.2 9" />
    </svg>
  );
}

function CogIcon() {
  return (
    <svg fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24" className="w-full h-full">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  );
}

function BarChartIcon() {
  return (
    <svg fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24" className="w-full h-full">
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 3v18h18M7 16v-4m4 4V8m4 8V5" />
    </svg>
  );
}

function LogoutIcon() {
  return (
    <svg fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24" className="w-full h-full">
      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
    </svg>
  );
}

// ── Nav ────────────────────────────────────────────────────────────────────────

export async function Nav() {
  const session = await auth();
  const hasUser = !!session?.user?.email;

  const signOutAction = async () => {
    "use server";
    await signOut({ redirectTo: "/login" });
  };

  return (
    <>
      {/* ── Mobile: top bar ────────────────────────────────────────────── */}
      <header className="md:hidden fixed top-0 inset-x-0 z-40 h-11 flex items-center justify-between px-4 bg-white/95 dark:bg-zinc-950/95 backdrop-blur-sm border-b border-zinc-200 dark:border-zinc-800">
        <span className="text-sm font-bold text-zinc-900 dark:text-zinc-50 tracking-tight">
          Stock Dash
        </span>
        <div className="flex items-center gap-1">
          <NavItem href="/settings" icon={<CogIcon />} label="Pengaturan" variant="mobile-topbar" />
          {hasUser && (
            <form action={signOutAction}>
              <button
                type="submit"
                title="Keluar"
                className="w-9 h-9 flex items-center justify-center rounded-lg text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
              >
                <span className="w-5 h-5"><LogoutIcon /></span>
              </button>
            </form>
          )}
        </div>
      </header>

      {/* Spacer: pushes page content below mobile top bar */}
      <div className="md:hidden h-11 flex-shrink-0" aria-hidden="true" />

      {/* ── Mobile: bottom nav ─────────────────────────────────────────── */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 z-40 h-14 flex items-center justify-around bg-white/95 dark:bg-zinc-950/95 backdrop-blur-sm border-t border-zinc-200 dark:border-zinc-800">
        <NavItem href="/watchlist" icon={<ClipboardIcon />} label="Watchlist" variant="mobile" />
        <NavItem href="/sectors" icon={<BarChartIcon />} label="Sektor" variant="mobile" />
        <NavItem href="/markets" icon={<GlobeIcon />} label="Markets" variant="mobile" />
      </nav>

      {/* ── Desktop: sidebar ───────────────────────────────────────────── */}
      <aside className="hidden md:flex fixed inset-y-0 left-0 z-40 w-14 flex-col bg-white dark:bg-zinc-950 border-r border-zinc-200 dark:border-zinc-800">
        {/* Brand */}
        <div className="h-14 flex items-center justify-center border-b border-zinc-200 dark:border-zinc-800">
          <span className="text-[10px] font-bold text-zinc-400 dark:text-zinc-600 tracking-widest uppercase select-none">
            SD
          </span>
        </div>

        {/* Main nav */}
        <div className="flex flex-col gap-1 p-2 flex-1 mt-2">
          <NavItem href="/watchlist" icon={<ClipboardIcon />} label="Watchlist" variant="sidebar" />
          <NavItem href="/sectors" icon={<BarChartIcon />} label="Sektor US" variant="sidebar" />
          <NavItem href="/markets" icon={<GlobeIcon />} label="Markets" variant="sidebar" />
        </div>

        {/* Bottom: Settings + Logout */}
        <div className="flex flex-col gap-1 p-2 border-t border-zinc-200 dark:border-zinc-800">
          <NavItem href="/settings" icon={<CogIcon />} label="Pengaturan" variant="sidebar" />
          {hasUser && (
            <form action={signOutAction}>
              <button
                type="submit"
                title="Keluar"
                className="flex items-center justify-center h-10 w-full rounded-lg text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
              >
                <span className="w-5 h-5"><LogoutIcon /></span>
              </button>
            </form>
          )}
        </div>
      </aside>
    </>
  );
}
