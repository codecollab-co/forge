"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type Branches } from "@/lib/api";

export default function NewPullPage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string }>();
  const [branches, setBranches] = useState<Branches | null>(null);
  const [base, setBase] = useState("");
  const [head, setHead] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        const b = await api.getBranches(params.owner, params.name);
        setBranches(b);
        setBase(b.default || (b.branches?.[0] ?? ""));
        const headGuess = (b.branches ?? []).find((br) => br !== b.default) ?? "";
        setHead(headGuess);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name, router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!title.trim() || !head || !base || head === base) {
      setError("Title required, head and base must differ.");
      return;
    }
    setBusy(true);
    try {
      const pr = await api.createPull(params.owner, params.name, {
        title,
        body,
        head_branch: head,
        base_branch: base,
      });
      router.push(`/${params.owner}/${params.name}/pulls/${pr.number}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  const opts = branches?.branches ?? [];

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">New Pull Request</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium">Base (merge into)</label>
            <select
              value={base}
              onChange={(e) => setBase(e.target.value)}
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            >
              {opts.map((b) => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium">Head (from)</label>
            <select
              value={head}
              onChange={(e) => setHead(e.target.value)}
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            >
              {opts.map((b) => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
        </div>
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
          <label className="block text-sm font-medium">Body (optional)</label>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={6}
            className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
          />
        </div>
        {error && <p className="text-red-600">Error: {error}</p>}
        <button
          type="submit"
          disabled={busy}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        >
          {busy ? "Creating…" : "Create Pull Request"}
        </button>
      </form>
    </main>
  );
}
