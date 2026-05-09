"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, type IssueDetail } from "@/lib/api";

export default function IssueDetailPage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string; number: string }>();
  const number = parseInt(params.number, 10);
  const [detail, setDetail] = useState<IssueDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  async function reload() {
    setDetail(await api.getIssue(params.owner, params.name, number));
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

  async function ensureAuth(): Promise<boolean> {
    if (await Session.doesSessionExist()) return true;
    router.replace("/auth");
    return false;
  }

  async function postComment(e: React.FormEvent) {
    e.preventDefault();
    if (!comment.trim() || !(await ensureAuth())) return;
    setBusy(true);
    setError(null);
    try {
      await api.addIssueComment(params.owner, params.name, number, comment);
      setComment("");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function toggleState() {
    if (!detail || !(await ensureAuth())) return;
    setBusy(true);
    setError(null);
    try {
      if (detail.issue.state === "open") {
        await api.closeIssue(params.owner, params.name, number);
      } else {
        await api.reopenIssue(params.owner, params.name, number);
      }
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  if (!detail) return <main className="mx-auto max-w-3xl px-6 py-12">Loading…</main>;
  const iss = detail.issue;

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-12">
      <header className="space-y-1">
        <p className="text-xs uppercase tracking-wide text-zinc-500">{iss.state}</p>
        <h1 className="text-2xl font-semibold">
          #{iss.number} {iss.title}
        </h1>
        <p className="text-sm text-zinc-500">
          opened by {iss.author} · {new Date(iss.created_at).toLocaleString()}
          {iss.assignee && ` · assigned to ${iss.assignee.handle}`}
        </p>
      </header>

      {iss.body && (
        <article className="prose prose-zinc max-w-none rounded-md border border-zinc-200 p-6 dark:prose-invert dark:border-zinc-800">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{iss.body}</ReactMarkdown>
        </article>
      )}

      {error && <p className="text-red-600">Error: {error}</p>}

      <button
        onClick={toggleState}
        disabled={busy}
        className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
      >
        {iss.state === "open" ? "Close issue" : "Reopen issue"}
      </button>

      <section className="space-y-4">
        <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">Comments</h2>
        {detail.comments.length === 0 && <p className="text-sm text-zinc-500">No comments yet.</p>}
        {detail.comments.map((c) => (
          <article key={c.id} className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
            <p className="mb-2 text-xs uppercase tracking-wide text-zinc-500">
              {c.author} · {new Date(c.created_at).toLocaleString()}
            </p>
            <div className="prose prose-sm prose-zinc max-w-none dark:prose-invert">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{c.body}</ReactMarkdown>
            </div>
          </article>
        ))}
        <form onSubmit={postComment} className="space-y-2">
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={3}
            className="w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            placeholder="Leave a comment (Markdown)"
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
