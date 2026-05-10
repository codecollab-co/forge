"use client";

import { useEffect, useRef, useState } from "react";
import { CopyButton } from "./CopyButton";

export function CodeDropdown({ cloneURL }: { cloneURL: string }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="rounded-md bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700"
      >
        Code <span className="ml-1 text-xs">▾</span>
      </button>
      {open && (
        <div className="absolute right-0 z-10 mt-1 w-96 rounded-md border border-zinc-200 bg-white p-3 shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-zinc-500">
            Clone HTTPS
          </p>
          <div className="flex items-center gap-2">
            <input
              readOnly
              value={cloneURL}
              onFocus={(e) => e.currentTarget.select()}
              className="flex-1 rounded-md border border-zinc-300 px-2 py-1 font-mono text-xs dark:border-zinc-700 dark:bg-zinc-950"
            />
            <CopyButton text={cloneURL} />
          </div>
          <p className="mt-2 text-xs text-zinc-500">
            Use a <a href="/me/tokens" className="underline">personal access token</a> as the password on first push.
          </p>
        </div>
      )}
    </div>
  );
}
