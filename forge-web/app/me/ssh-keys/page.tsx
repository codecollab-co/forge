"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type SSHKey } from "@/lib/api";

export default function SSHKeysPage() {
  const router = useRouter();
  const [keys, setKeys] = useState<SSHKey[]>([]);
  const [error, setError] = useState<string | null>(null);

  const [showNew, setShowNew] = useState(false);
  const [name, setName] = useState("");
  const [publicKey, setPublicKey] = useState("");
  const [busy, setBusy] = useState(false);

  async function reload() {
    setKeys(await api.listSSHKeys() ?? []);
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

  async function add(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || !publicKey.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await api.addSSHKey(name.trim(), publicKey.trim());
      setName("");
      setPublicKey("");
      setShowNew(false);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function revoke(id: string, label: string) {
    if (!confirm(`Revoke "${label}"? Any host using it loses access immediately.`)) return;
    setBusy(true);
    setError(null);
    try {
      await api.revokeSSHKey(id);
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
          <h1 className="text-2xl font-semibold">SSH keys</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Authorise machines to <code>git clone</code> / <code>git push</code> via SSH.
          </p>
        </div>
        {!showNew && (
          <button
            onClick={() => setShowNew(true)}
            className="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
          >
            + New key
          </button>
        )}
      </div>

      {error && <p className="text-red-600">Error: {error}</p>}

      {showNew && (
        <form onSubmit={add} className="space-y-3 rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
          <div>
            <label className="block text-sm font-medium">Key name</label>
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="laptop, work-mac, …"
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Public key</label>
            <textarea
              value={publicKey}
              onChange={(e) => setPublicKey(e.target.value)}
              rows={4}
              placeholder="ssh-ed25519 AAAA… you@machine"
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 font-mono text-xs dark:border-zinc-700 dark:bg-zinc-900"
              required
            />
            <p className="mt-1 text-xs text-zinc-500">
              Paste the contents of <code>~/.ssh/id_ed25519.pub</code> (or
              <code>~/.ssh/id_rsa.pub</code>).
            </p>
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={busy || !name.trim() || !publicKey.trim()}
              className="rounded-md bg-zinc-900 px-3 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
            >
              {busy ? "Adding…" : "Add key"}
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

      {keys.length === 0 ? (
        <p className="rounded-md border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
          No SSH keys yet. Add one above to clone over SSH.
        </p>
      ) : (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
          {keys.map((k) => (
            <li key={k.id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
              <div className="min-w-0 flex-1">
                <p className="font-medium">{k.name}</p>
                <p className="font-mono text-xs text-zinc-500">{k.fingerprint}</p>
                <p className="text-xs text-zinc-500">
                  added {new Date(k.created_at).toLocaleDateString()}
                  {k.last_used_at && ` · last used ${new Date(k.last_used_at).toLocaleDateString()}`}
                </p>
              </div>
              <button
                onClick={() => revoke(k.id, k.name)}
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
