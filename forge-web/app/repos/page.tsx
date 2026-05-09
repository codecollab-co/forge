"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type Repo } from "@/lib/api";

export default function ReposPage() {
  const router = useRouter();
  const [repos, setRepos] = useState<Repo[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        setRepos(await api.listRepos());
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-16">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Your Repositories</h1>
        <Link
          href="/repos/new"
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
        >
          New Repository
        </Link>
      </div>

      {error && <p className="text-red-600">Error: {error}</p>}

      {repos && repos.length === 0 && (
        <p className="text-zinc-600 dark:text-zinc-400">
          No repositories yet. Create your first one.
        </p>
      )}

      {repos && repos.length > 0 && (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
          {repos.map((r) => (
            <li key={r.id} className="p-4">
              <Link
                href={`/${r.owner}/${r.name}`}
                className="text-lg font-medium hover:underline"
              >
                {r.owner}/{r.name}
              </Link>
              {r.description && (
                <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                  {r.description}
                </p>
              )}
              <p className="mt-1 text-xs uppercase tracking-wide text-zinc-400">
                {r.visibility}
              </p>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
