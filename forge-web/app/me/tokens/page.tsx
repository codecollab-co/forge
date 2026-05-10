"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { CopyButton } from "@/app/components/CopyButton";
import { api, type TokenSummary } from "@/lib/api";

export default function TokensPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [tokens, setTokens] = useState<TokenSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  const [showNew, setShowNew] = useState(false);
  const [name, setName] = useState("");
  const [expiresDays, setExpiresDays] = useState<number | "">(90);
  const [busy, setBusy] = useState(false);
  const [revealed, setRevealed] = useState<{ name: string; secret: string } | null>(null);

  async function reload() {
    const r = await api.listTokens();
    setUsername(r.username);
    setTokens(r.tokens || []);
  }

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        await reload();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  async function mint(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const out = await api.mintToken(name.trim(), expiresDays === "" ? 0 : Number(expiresDays));
      setRevealed({ name: out.token.name, secret: out.secret });
      setName("");
      setShowNew(false);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function revoke(id: string, label: string) {
    if (!confirm(`Revoke "${label}"? Anything using it will stop working immediately.`)) return;
    setBusy(true);
    setError(null);
    try {
      await api.revokeToken(id);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-10">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Personal access tokens</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Used as your <code>git push</code> password and by the <code>forge</code> CLI.
          </p>
        </div>
        {!showNew && (
          <button
            onClick={() => setShowNew(true)}
            className="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
          >
            + New token
          </button>
        )}
      </div>

      {error && <p className="text-red-600">Error: {error}</p>}

      {revealed && (
        <section className="space-y-3 rounded-md border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950">
          <p className="text-sm font-medium">
            Copy <code>{revealed.name}</code> now — it won't be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded-md border border-amber-200 bg-white px-3 py-2 font-mono text-sm dark:border-amber-800 dark:bg-zinc-900">
              {revealed.secret}
            </code>
            <CopyButton text={revealed.secret} label="Copy" />
          </div>
          <details className="text-xs text-zinc-600 dark:text-zinc-400">
            <summary className="cursor-pointer">How do I use this?</summary>
            <p className="mt-2">
              <strong>git:</strong> use it as the password when prompted on push/clone. Username is{" "}
              <code>{username}</code>.
            </p>
            <p className="mt-1">
              <strong>forge CLI:</strong> use <code>forge auth login</code> instead — the device-code flow handles tokens for you.
            </p>
          </details>
        </section>
      )}

      {showNew && (
        <form onSubmit={mint} className="space-y-3 rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
          <div>
            <label className="block text-sm font-medium">Token name</label>
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="laptop, ci-runner, ..."
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Expires in (days)</label>
            <input
              type="number"
              min={0}
              value={expiresDays}
              onChange={(e) => setExpiresDays(e.target.value === "" ? "" : Number(e.target.value))}
              className="mt-1 w-32 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
            />
            <p className="mt-1 text-xs text-zinc-500">0 = never expires.</p>
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={busy || !name.trim()}
              className="rounded-md bg-zinc-900 px-3 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
            >
              {busy ? "Minting…" : "Generate token"}
            </button>
            <button
              type="button"
              onClick={() => setShowNew(false)}
              className="rounded-md border border-zinc-300 px-3 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {tokens.length === 0 ? (
        <p className="rounded-md border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
          No tokens yet. Mint one above to use git over HTTPS.
        </p>
      ) : (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
          {tokens.map((t) => (
            <li key={t.id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
              <div>
                <p className="font-medium">{t.name}</p>
                <p className="text-xs text-zinc-500">
                  created {new Date(t.created_at).toLocaleDateString()}
                  {t.last_used_at && ` · last used ${new Date(t.last_used_at).toLocaleDateString()}`}
                  {t.expires_at && ` · expires ${new Date(t.expires_at).toLocaleDateString()}`}
                </p>
              </div>
              <button
                onClick={() => revoke(t.id, t.name)}
                disabled={busy}
                className="rounded-md border border-red-300 px-3 py-1 text-xs text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
              >
                Revoke
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
