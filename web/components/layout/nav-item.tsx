"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

interface NavItemProps {
  href: string;
  icon: React.ReactNode;
  label: string;
  variant: "sidebar" | "mobile" | "mobile-topbar";
}

export function NavItem({ href, icon, label, variant }: NavItemProps) {
  const pathname = usePathname();
  const active = pathname === href || pathname.startsWith(href + "/");

  if (variant === "sidebar") {
    return (
      <Link
        href={href}
        title={label}
        className={[
          "flex items-center justify-center h-10 w-full rounded-lg transition-colors",
          active
            ? "bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-50"
            : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800",
        ].join(" ")}
      >
        <span className="w-5 h-5 flex-shrink-0">{icon}</span>
      </Link>
    );
  }

  if (variant === "mobile-topbar") {
    return (
      <Link
        href={href}
        title={label}
        className={[
          "w-9 h-9 flex items-center justify-center rounded-lg transition-colors",
          active
            ? "text-zinc-900 dark:text-zinc-50 bg-zinc-100 dark:bg-zinc-800"
            : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-50 hover:bg-zinc-100 dark:hover:bg-zinc-800",
        ].join(" ")}
      >
        <span className="w-5 h-5 flex-shrink-0">{icon}</span>
      </Link>
    );
  }

  return (
    <Link
      href={href}
      className={[
        "flex flex-col items-center gap-0.5 py-1 px-5 transition-colors",
        active
          ? "text-blue-600 dark:text-blue-400"
          : "text-zinc-500 dark:text-zinc-400",
      ].join(" ")}
    >
      <span className="w-6 h-6 flex-shrink-0">{icon}</span>
      <span className="text-[10px] font-medium">{label}</span>
    </Link>
  );
}
