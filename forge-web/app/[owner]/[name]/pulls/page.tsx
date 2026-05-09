"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type PRState, type PullRequest } from "@/lib/api";

export default function PullsPage() {
  const params = useParams<{ owner: string; name: string }>();
  const [state, setState] = useState<PRState>("open");
  const [list, setList] = useState<PullRequest[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setList(await api.listPulls(params.owner, params.name, state));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name, state]);

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-12">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">
          Pull Requests · {params.owner}/{params.name}
        </h1>
        <Link
          href={`/${params.owner}/${params.name}/pulls/new`}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
        >
          New PR
        </Link>
      </div>

      <div className="flex gap-2 text-sm">
        {(["open", "merged", "closed"] as PRState[]).map((s) => (
          <button
            key={s}
            onClick={() => setState(s)}
            className={
              s === state
                ? "rounded-md bg-zinc-900 px-3 py-1 text-white dark:bg-zinc-100 dark:text-zinc-900"
                : "rounded-md border border-zinc-300 px-3 py-1 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
            }
          >
            {s}
          </button>
        ))}
      </div>

      {error && <p className="text-red-600">Error: {error}</p>}
      {list && list.length === 0 && <p className="text-zinc-500">No {state} pull requests.</p>}
      {list && list.length > 0 && (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
          {list.map((pr) => (
            <li key={pr.id} className="p-4">
              <Link
                href={`/${params.owner}/${params.name}/pulls/${pr.number}`}
                className="text-lg font-medium hover:underline"
              >
                #{pr.number} {pr.title}
              </Link>
              <p className="mt-1 text-xs text-zinc-500">
                {pr.author} · {pr.head_branch} → {pr.base_branch} · {pr.state}
              </p>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
