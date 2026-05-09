"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type Commit } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";

export default function CommitsPage() {
  const params = useParams<{ owner: string; name: string; branch: string }>();
  const branch = decodeURIComponent(params.branch);
  const [commits, setCommits] = useState<Commit[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setCommits(await api.listCommits(params.owner, params.name, branch));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name, branch]);

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-4xl space-y-4 px-6 py-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold">
            Commits on <code>{branch}</code>
          </h1>
          <Link
            href={branch === "main" ? `/${params.owner}/${params.name}` : `/${params.owner}/${params.name}?branch=${encodeURIComponent(branch)}`}
            className="rounded-md border border-zinc-300 px-3 py-1 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
          >
            Back to code
          </Link>
        </div>
        {error && <p className="text-red-600">Error: {error}</p>}
        {commits && commits.length === 0 && (
          <p className="text-sm text-zinc-500">No commits on this branch yet.</p>
        )}
        {commits && commits.length > 0 && (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {commits.map((c) => (
              <li key={c.oid} className="flex items-start justify-between gap-3 px-4 py-3 text-sm">
                <div className="min-w-0 flex-1">
                  <Link
                    href={`/${params.owner}/${params.name}/commit/${c.oid}`}
                    className="font-medium hover:underline"
                  >
                    {c.subject}
                  </Link>
                  <p className="mt-1 text-xs text-zinc-500">
                    {c.author_name} · {new Date(c.author_date).toLocaleString()}
                  </p>
                </div>
                <Link
                  href={`/${params.owner}/${params.name}/commit/${c.oid}`}
                  className="rounded-md border border-zinc-300 px-2 py-1 font-mono text-xs hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
                >
                  {c.short_oid}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </main>
    </>
  );
}
