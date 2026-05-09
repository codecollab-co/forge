"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type PullDetail } from "@/lib/api";

export default function PullDetailPage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string; number: string }>();
  const number = parseInt(params.number, 10);
  const [detail, setDetail] = useState<PullDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  async function reload() {
    setDetail(await api.getPull(params.owner, params.name, number));
  }

  useEffect(() => {
    (async () => {
      try {
        await reload();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.owner, params.name, number]);

  async function postComment(e: React.FormEvent) {
    e.preventDefault();
    if (!comment.trim()) return;
    if (!(await Session.doesSessionExist())) {
      router.replace("/auth");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await api.addPullComment(params.owner, params.name, number, comment);
      setComment("");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function merge() {
    if (!(await Session.doesSessionExist())) {
      router.replace("/auth");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await api.mergePull(params.owner, params.name, number);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  if (!detail) return <main className="mx-auto max-w-3xl px-6 py-12">Loading…</main>;
  const pr = detail.pull_request;

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-12">
      <header className="space-y-1">
        <p className="text-xs uppercase tracking-wide text-zinc-500">{pr.state}</p>
        <h1 className="text-2xl font-semibold">
          #{pr.number} {pr.title}
        </h1>
        <p className="text-sm text-zinc-500">
          {pr.author} wants to merge <code>{pr.head_branch}</code> into <code>{pr.base_branch}</code>
        </p>
        {pr.body && <p className="mt-3 whitespace-pre-wrap">{pr.body}</p>}
      </header>

      {error && <p className="text-red-600">Error: {error}</p>}

      {pr.state === "open" && (
        <button
          onClick={merge}
          disabled={busy}
          className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          {busy ? "Merging…" : "Merge pull request"}
        </button>
      )}

      <section className="rounded-md border border-zinc-200 dark:border-zinc-800">
        <h2 className="border-b border-zinc-200 px-4 py-2 text-sm font-medium uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
          Diff
        </h2>
        <pre className="overflow-x-auto p-4 text-xs">{detail.diff || "(no changes)"}</pre>
      </section>

      <section className="space-y-4">
        <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">Comments</h2>
        {detail.comments.length === 0 && <p className="text-sm text-zinc-500">No comments yet.</p>}
        {detail.comments.map((c) => (
          <article key={c.id} className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
            <p className="mb-2 text-xs uppercase tracking-wide text-zinc-500">
              {c.author} {c.author_kind === "agent" && "· agent"} ·{" "}
              {new Date(c.created_at).toLocaleString()}
            </p>
            <p className="whitespace-pre-wrap">{c.body}</p>
          </article>
        ))}
        <form onSubmit={postComment} className="space-y-2">
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={3}
            className="w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            placeholder="Leave a comment"
          />
          <button
            type="submit"
            disabled={busy || !comment.trim()}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          >
            Comment
          </button>
        </form>
      </section>
    </main>
  );
}
