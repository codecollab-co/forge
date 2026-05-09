"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type CommitDetail } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";

export default function CommitPage() {
  const params = useParams<{ owner: string; name: string; oid: string }>();
  const [c, setC] = useState<CommitDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setC(await api.getCommit(params.owner, params.name, params.oid));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name, params.oid]);

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-4xl space-y-4 px-6 py-8">
        {error && <p className="text-red-600">Error: {error}</p>}
        {c && (
          <>
            <header className="space-y-1">
              <p className="font-mono text-xs text-zinc-500">{c.oid}</p>
              <h1 className="text-2xl font-semibold">{c.subject}</h1>
              <p className="text-sm text-zinc-500">
                {c.author_name} &lt;{c.author_email}&gt; · {new Date(c.author_date).toLocaleString()}
              </p>
              {c.parents.length > 0 && (
                <p className="text-xs text-zinc-500">
                  Parents:{" "}
                  {c.parents.map((p, i) => (
                    <span key={p}>
                      {i > 0 && ", "}
                      <Link
                        href={`/${params.owner}/${params.name}/commit/${p}`}
                        className="font-mono underline"
                      >
                        {p.slice(0, 7)}
                      </Link>
                    </span>
                  ))}
                </p>
              )}
            </header>

            <section className="rounded-md border border-zinc-200 dark:border-zinc-800">
              <h2 className="border-b border-zinc-200 px-4 py-2 text-sm font-medium uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
                Diff
              </h2>
              <pre className="overflow-x-auto p-4 text-xs">{c.diff || "(empty diff)"}</pre>
            </section>
          </>
        )}
      </main>
    </>
  );
}
