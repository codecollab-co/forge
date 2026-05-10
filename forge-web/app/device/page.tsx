"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { apiDomain } from "@/lib/supertokens";

export default function DevicePage() {
  const router = useRouter();
  const sp = useSearchParams();
  const [code, setCode] = useState((sp.get("user_code") || "").toUpperCase());
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        const back = encodeURIComponent(`/device${code ? `?user_code=${code}` : ""}`);
        router.replace(`/auth?redirectToPath=${back}`);
      }
    })();
  }, [router, code]);

  async function approve(e: React.FormEvent) {
    e.preventDefault();
    if (!code.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`${apiDomain}/oauth/device/approve`, {
        method: "POST",
        credentials: "include",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ user_code: code.trim() }),
      });
      if (!res.ok) {
        setError(await res.text());
        return;
      }
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-md space-y-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Authorize device</h1>
      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        Enter the code shown in your terminal.
      </p>

      {!done ? (
        <form onSubmit={approve} className="space-y-3">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="XXXX-XXXX"
            autoFocus
            required
            pattern="[A-Z2-9]{4}-[A-Z2-9]{4}"
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-center font-mono text-lg tracking-widest dark:border-zinc-700 dark:bg-zinc-900"
          />
          {error && <p className="text-red-600">Error: {error}</p>}
          <button
            type="submit"
            disabled={busy || !code.trim()}
            className="w-full rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          >
            {busy ? "Authorizing…" : "Authorize"}
          </button>
        </form>
      ) : (
        <div className="rounded-md border border-emerald-300 bg-emerald-50 p-4 text-center dark:border-emerald-700 dark:bg-emerald-950">
          <p className="font-medium">Device authorized.</p>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Return to your terminal — your CLI is signing in.
          </p>
        </div>
      )}
    </main>
  );
}
