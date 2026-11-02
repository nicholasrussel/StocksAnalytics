"use client";

import { useState, useRef, useEffect } from "react";

interface TooltipProps {
  content: React.ReactNode;
  children: React.ReactNode;
}

export function Tooltip({ content, children }: TooltipProps) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<"top" | "bottom">("top");
  const triggerRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open || !triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    setPos(rect.top > 120 ? "top" : "bottom");
  }, [open]);

  return (
    <span className="relative inline-flex items-center">
      <span
        ref={triggerRef}
        className="cursor-help underline decoration-dotted decoration-zinc-400 dark:decoration-zinc-600 underline-offset-2"
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        tabIndex={0}
      >
        {children}
      </span>

      {open && (
        <span
          className={`
            absolute z-50 left-0 w-64 rounded-lg border border-zinc-200 dark:border-zinc-700
            bg-white dark:bg-zinc-900 shadow-lg px-3 py-2.5 text-xs leading-relaxed
            text-zinc-700 dark:text-zinc-300 pointer-events-none
            ${pos === "top" ? "bottom-full mb-2" : "top-full mt-2"}
          `}
        >
          {content}
        </span>
      )}
    </span>
  );
}
