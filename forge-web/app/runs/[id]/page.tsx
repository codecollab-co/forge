"use client";

import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { api, type Run } from "@/lib/api";

const AGENT_API_URL =
  process.env.NEXT_PUBLIC_AGENT_API_URL ?? "http://localhost:8081";

type Trace = { id?: number; type: string; payload: Record<string, unknown> };

export default function RunPage() {
  const params = useParams<{ id: string }>();
  const [run, setRun] = useState<Run | null>(null);
  const [trace, setTrace] = useState<Trace[]>([]);
  const [error, setError] = useState<string | null>(null);
  const sseRef = useRef<EventSource | null>(null);
  const traceEndRef = useRef<HTMLDivElement | null>(null);

  // Initial Run state.
  useEffect(() => {
    (async () => {
      try {
        setRun(await api.getRun(params.id));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.id]);

  // SSE stream.
  useEffect(() => {
    const es = new EventSource(`${AGENT_API_URL}/runs/${params.id}/stream`);
    sseRef.current = es;
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const evt: Trace = { id: e.lastEventId ? Number(e.lastEventId) : undefined, type: data.type, payload: data.payload };
        setTrace((t) => [...t, evt]);
        if (evt.type === "run.terminal") {
          api.getRun(params.id).then(setRun).catch(() => {});
        }
      } catch {
        /* ignore non-JSON pings */
      }
    };
    es.onerror = () => {
      // EventSource auto-reconnects with Last-Event-ID. No-op.
    };
    return () => {
      es.close();
      sseRef.current = null;
    };
  }, [params.id]);

  // Auto-scroll to bottom.
  useEffect(() => {
    traceEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [trace.length]);

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-12">
      <header className="space-y-1">
        <p className="text-xs uppercase tracking-wide text-zinc-500">Agent run</p>
        <h1 className="font-mono text-lg">{params.id.slice(0, 8)}</h1>
        {run && (
          <p className="text-sm text-zinc-500">
            state: <code>{run.state}</code>
            {run.cancel_requested && run.state === "running" && " · cancelling"}
            {run.pr_number && (
              <>
                {" · "}
                <Link className="underline" href="#">PR #{run.pr_number}</Link>
              </>
            )}
          </p>
        )}
      </header>

      {error && <p className="text-red-600">Error: {error}</p>}

      <section className="rounded-md border border-zinc-200 dark:border-zinc-800">
        <h2 className="border-b border-zinc-200 px-4 py-2 text-sm font-medium uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
          Live trace
        </h2>
        <ol className="divide-y divide-zinc-200 dark:divide-zinc-800">
          {trace.length === 0 && (
            <li className="px-4 py-3 text-sm text-zinc-500">Waiting for events…</li>
          )}
          {trace.map((evt, i) => (
            <li key={`${evt.id ?? "live"}-${i}`} className="px-4 py-2 text-sm">
              <p className="font-mono text-xs text-zinc-500">{evt.type}</p>
              <pre className="mt-1 overflow-x-auto whitespace-pre-wrap text-xs">
                {JSON.stringify(evt.payload, null, 2)}
              </pre>
            </li>
          ))}
        </ol>
        <div ref={traceEndRef} />
      </section>
    </main>
  );
}
