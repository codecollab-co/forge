"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export function BranchSwitcher({
  owner,
  name,
  current,
  branches,
  defaultBranch,
  canCreate,
}: {
  owner: string;
  name: string;
  current: string;
  branches: string[];
  defaultBranch: string;
  canCreate: boolean;
}) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setError(null);
      }
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const trimmed = filter.trim();
  const filtered = trimmed
    ? branches.filter((b) => b.toLowerCase().includes(trimmed.toLowerCase()))
    : branches;
  const exactMatch = branches.includes(trimmed);
  const showCreate = canCreate && trimmed.length > 0 && !exactMatch && /^[A-Za-z0-9._/-]+$/.test(trimmed);

  function navigate(branch: string) {
    setOpen(false);
    setFilter("");
    if (branch === defaultBranch) {
      router.push(`/${owner}/${name}`);
    } else {
      router.push(`/${owner}/${name}?branch=${encodeURIComponent(branch)}`);
    }
  }

  async function createAndSwitch() {
    if (!showCreate) return;
    setCreating(true);
    setError(null);
    try {
      await api.createBranch(owner, name, { name: trimmed, from: current });
      navigate(trimmed);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setCreating(false);
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-950 dark:hover:bg-zinc-900"
      >
        <span className="text-zinc-500">Branch:</span>
        <span className="font-medium">{current}</span>
        <span className="text-xs">▾</span>
      </button>
      {open && (
        <div className="absolute z-10 mt-1 w-72 rounded-md border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
          <div className="border-b border-zinc-200 p-2 dark:border-zinc-800">
            <input
              autoFocus
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Find or create a branch…"
              className="w-full rounded-md border border-zinc-300 px-2 py-1 text-sm dark:border-zinc-700 dark:bg-zinc-950"
            />
          </div>
          <ul className="max-h-64 overflow-y-auto py-1">
            {filtered.length === 0 && trimmed === "" && (
              <li className="px-3 py-2 text-xs text-zinc-500">No branches yet.</li>
            )}
            {filtered.map((b) => (
              <li key={b}>
                <button
                  onClick={() => navigate(b)}
                  className="flex w-full items-center justify-between px-3 py-1.5 text-left text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800"
                >
                  <span className="truncate">{b}</span>
                  {b === current && <span className="ml-2 text-xs text-emerald-600">current</span>}
                  {b === defaultBranch && b !== current && (
                    <span className="ml-2 text-xs text-zinc-400">default</span>
                  )}
                </button>
              </li>
            ))}
          </ul>
          {showCreate && (
            <div className="border-t border-zinc-200 p-2 dark:border-zinc-800">
              <button
                onClick={createAndSwitch}
                disabled={creating}
                className="w-full rounded-md bg-zinc-900 px-3 py-1.5 text-left text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
              >
                {creating ? "Creating…" : (
                  <>Create branch: <code>{trimmed}</code> from <code>{current}</code></>
                )}
              </button>
              {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
            </div>
          )}
          <div className="border-t border-zinc-200 dark:border-zinc-800">
            <Link
              href={`/${owner}/${name}/branches`}
              className="block px-3 py-2 text-xs text-zinc-600 hover:bg-zinc-50 dark:text-zinc-400 dark:hover:bg-zinc-800"
              onClick={() => setOpen(false)}
            >
              View all branches →
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
