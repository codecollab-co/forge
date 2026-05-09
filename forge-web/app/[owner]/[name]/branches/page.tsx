"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type Branches, type Me } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";

export default function BranchesPage() {
  const params = useParams<{ owner: string; name: string }>();
  const [me, setMe] = useState<Me | null>(null);
  const [info, setInfo] = useState<Branches | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [newName, setNewName] = useState("");
  const [busy, setBusy] = useState(false);

  async function reload() {
    setInfo(await api.getBranches(params.owner, params.name));
  }

  useEffect(() => {
    (async () => {
      try {
        const [m, b] = await Promise.all([api.me().catch(() => null), api.getBranches(params.owner, params.name)]);
        setMe(m);
        setInfo(b);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name]);

  const isOwner = me?.handle === params.owner;
  const branches = info?.branches ?? [];
  const defaultBranch = info?.default ?? "main";

  async function createBranch(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.createBranch(params.owner, params.name, { name: newName, from: defaultBranch });
      setNewName("");
      setShowNew(false);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function rename(branch: string) {
    const newN = window.prompt(`Rename branch ${branch} to:`, branch);
    if (!newN || newN === branch) return;
    setBusy(true);
    setError(null);
    try {
      await api.renameBranch(params.owner, params.name, branch, newN);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function destroy(branch: string) {
    if (!window.confirm(`Delete branch ${branch}? Commits unique to this branch will be unreachable.`)) return;
    setBusy(true);
    setError(null);
    try {
      await api.deleteBranch(params.owner, params.name, branch);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-4xl space-y-6 px-6 py-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Branches</h1>
          {isOwner && (
            <button
              onClick={() => setShowNew((v) => !v)}
              className="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
            >
              + New branch
            </button>
          )}
        </div>

        {showNew && isOwner && (
          <form onSubmit={createBranch} className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
            <label className="block text-sm font-medium">New branch name</label>
            <div className="mt-1 flex gap-2">
              <input
                autoFocus
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="my-feature"
                className="flex-1 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
              />
              <button
                type="submit"
                disabled={busy || !newName.trim()}
                className="rounded-md bg-zinc-900 px-3 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
              >
                Create
              </button>
            </div>
            <p className="mt-1 text-xs text-zinc-500">Branched from <code>{defaultBranch}</code>.</p>
          </form>
        )}

        {error && <p className="text-red-600">Error: {error}</p>}

        {branches.length === 0 ? (
          <p className="text-sm text-zinc-500">No branches yet. Push a first commit or create one above.</p>
        ) : (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {branches.map((b) => (
              <li key={b} className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="flex items-center gap-3">
                  <Link
                    href={b === defaultBranch ? `/${params.owner}/${params.name}` : `/${params.owner}/${params.name}?branch=${encodeURIComponent(b)}`}
                    className="font-medium hover:underline"
                  >
                    {b}
                  </Link>
                  {b === defaultBranch && (
                    <span className="rounded bg-emerald-200 px-2 py-0.5 text-xs text-emerald-900 dark:bg-emerald-900 dark:text-emerald-100">
                      default
                    </span>
                  )}
                </div>
                {isOwner && b !== defaultBranch && (
                  <div className="flex gap-2 text-xs">
                    <button
                      onClick={() => rename(b)}
                      disabled={busy}
                      className="rounded-md border border-zinc-300 px-2 py-1 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
                    >
                      Rename
                    </button>
                    <button
                      onClick={() => destroy(b)}
                      disabled={busy}
                      className="rounded-md border border-red-300 px-2 py-1 text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-400 dark:hover:bg-red-950"
                    >
                      Delete
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </main>
    </>
  );
}
