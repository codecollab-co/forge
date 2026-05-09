"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import Session from "supertokens-auth-react/recipe/session";
import { api, type DashboardRun, type Repo } from "@/lib/api";

const RUN_BADGE: Record<DashboardRun["state"], string> = {
  queued: "bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  running: "bg-violet-200 text-violet-900 dark:bg-violet-800 dark:text-violet-100",
  succeeded: "bg-emerald-200 text-emerald-900 dark:bg-emerald-800 dark:text-emerald-100",
  failed: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-100",
  cancelled: "bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
};

export default function Home() {
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [runs, setRuns] = useState<DashboardRun[] | null>(null);
  const [repos, setRepos] = useState<Repo[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      const has = await Session.doesSessionExist();
      setSignedIn(has);
      if (!has) return;
      try {
        const [r, rp] = await Promise.all([api.listMyRuns(), api.listRepos()]);
        setRuns(r);
        setRepos(rp);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, []);

  if (signedIn === null) {
    return <main className="mx-auto max-w-3xl px-6 py-16 text-zinc-500">Loading…</main>;
  }

  if (!signedIn) {
    return (
      <main className="mx-auto max-w-3xl space-y-6 px-6 py-24">
        <h1 className="text-4xl font-semibold tracking-tight">Forge</h1>
        <p className="text-lg text-zinc-600 dark:text-zinc-400">
          AI-native Git host. Assign an issue to an agent and a pull request appears.
        </p>
        <div className="flex gap-3 pt-4">
          <Link
            href="/auth"
            className="rounded-md bg-zinc-900 px-5 py-2 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
          >
            Sign in
          </Link>
        </div>
      </main>
    );
  }

  const active = (runs ?? []).filter((r) => r.state === "queued" || r.state === "running");
  const recent = (runs ?? []).filter((r) => r.state !== "queued" && r.state !== "running").slice(0, 10);

  return (
    <main className="mx-auto max-w-5xl space-y-8 px-6 py-10">
      {error && <p className="text-red-600">Error: {error}</p>}

      <section>
        <h2 className="mb-3 text-sm font-medium uppercase tracking-wide text-zinc-500">
          Active agent runs
        </h2>
        {active.length === 0 ? (
          <p className="rounded-md border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
            No active runs. Open an issue and click "Assign to Agent" to start one.
          </p>
        ) : (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {active.map((r) => (
              <RunRow key={r.id} run={r} />
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium uppercase tracking-wide text-zinc-500">
          Recent runs
        </h2>
        {recent.length === 0 ? (
          <p className="text-sm text-zinc-500">No runs yet.</p>
        ) : (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {recent.map((r) => (
              <RunRow key={r.id} run={r} />
            ))}
          </ul>
        )}
      </section>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">
            Your repositories
          </h2>
          <Link
            href="/repos/new"
            className="rounded-md bg-zinc-900 px-3 py-1.5 text-xs text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
          >
            New repository
          </Link>
        </div>
        {repos && repos.length === 0 && (
          <p className="rounded-md border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
            No repositories yet. Create your first one.
          </p>
        )}
        {repos && repos.length > 0 && (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            {repos.map((r) => (
              <li key={r.id} className="px-4 py-3 text-sm">
                <Link href={`/${r.owner}/${r.name}`} className="font-medium hover:underline">
                  {r.owner}/{r.name}
                </Link>
                {r.description && (
                  <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">{r.description}</p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}

function RunRow({ run }: { run: DashboardRun }) {
  return (
    <li className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
      <div className="min-w-0 flex-1">
        <Link
          href={`/${run.repo_owner}/${run.repo_name}/issues/${run.issue_number}`}
          className="truncate font-medium hover:underline"
        >
          {run.repo_owner}/{run.repo_name} · #{run.issue_number} {run.issue_title}
        </Link>
        <p className="mt-0.5 text-xs text-zinc-500">
          {new Date(run.created_at).toLocaleString()}
          {run.pr_number && (
            <>
              {" · "}
              <Link
                href={`/${run.repo_owner}/${run.repo_name}/pulls/${run.pr_number}`}
                className="underline"
              >
                PR #{run.pr_number}
              </Link>
            </>
          )}
        </p>
      </div>
      <span className={`rounded-md px-2 py-0.5 text-xs ${RUN_BADGE[run.state]}`}>{run.state}</span>
      <Link
        href={`/runs/${run.id}`}
        className="rounded-md border border-zinc-300 px-2 py-1 text-xs hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
      >
        Trace
      </Link>
    </li>
  );
}
