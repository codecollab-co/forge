"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api } from "@/lib/api";

export default function NewIssuePage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string }>();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) router.replace("/auth");
    })();
  }, [router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    setError(null);
    setBusy(true);
    try {
      const iss = await api.createIssue(params.owner, params.name, { title, body });
      router.push(`/${params.owner}/${params.name}/issues/${iss.number}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">New Issue</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium">Title</label>
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium">Body (Markdown)</label>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={10}
            className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 font-mono text-sm dark:border-zinc-700 dark:bg-zinc-900"
          />
        </div>
        {error && <p className="text-red-600">Error: {error}</p>}
        <button
          type="submit"
          disabled={busy || !title.trim()}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        >
          {busy ? "Creating…" : "Submit Issue"}
        </button>
      </form>
    </main>
  );
}
