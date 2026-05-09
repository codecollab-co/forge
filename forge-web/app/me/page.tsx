"use client";

import { useEffect, useState } from "react";
import Session from "supertokens-auth-react/recipe/session";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, type Me } from "@/lib/api";

export default function MePage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [draftName, setDraftName] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    (async () => {
      const has = await Session.doesSessionExist();
      if (!has) {
        router.replace("/auth");
        return;
      }
      try {
        const m = await api.me();
        setMe(m);
        setDraftName(m.display_name || "");
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!me) return;
    setSaving(true);
    setError(null);
    try {
      const updated = await api.updateMe({ display_name: draftName });
      setMe(updated);
      setSavedAt(Date.now());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  if (!me) {
    return <main className="mx-auto max-w-2xl px-6 py-16 text-zinc-500">Loading…</main>;
  }

  return (
    <main className="mx-auto max-w-2xl space-y-8 px-6 py-10">
      <header className="flex items-center gap-4">
        {me.avatar_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={me.avatar_url} alt="" className="h-16 w-16 rounded-full" />
        ) : (
          <span className="grid h-16 w-16 place-items-center rounded-full bg-zinc-300 text-xl font-semibold dark:bg-zinc-700">
            {me.handle.slice(0, 1).toUpperCase()}
          </span>
        )}
        <div>
          <p className="text-lg font-semibold">{me.display_name || me.handle}</p>
          <p className="text-sm text-zinc-500">@{me.handle}</p>
        </div>
      </header>

      {error && <p className="text-red-600">Error: {error}</p>}

      <section className="space-y-3 rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
        <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">Profile</h2>
        <form onSubmit={save} className="space-y-4">
          <div>
            <label className="block text-sm font-medium">Display name</label>
            <input
              value={draftName}
              onChange={(e) => setDraftName(e.target.value)}
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
              placeholder="How you appear in commits and comments"
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Email</label>
            <input
              value={me.email || ""}
              disabled
              className="mt-1 w-full rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900"
            />
            <p className="mt-1 text-xs text-zinc-500">
              Email is set by your sign-in method and can't be changed here.
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium">Handle</label>
            <input
              value={me.handle}
              disabled
              className="mt-1 w-full rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900"
            />
            <p className="mt-1 text-xs text-zinc-500">
              Used in repository URLs (e.g. <code>{me.handle}/your-repo</code>). Renames coming later.
            </p>
          </div>
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          >
            {saving ? "Saving…" : "Save"}
          </button>
          {savedAt && <span className="ml-3 text-xs text-emerald-600">Saved.</span>}
        </form>
      </section>

      <section className="rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
        <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">Access</h2>
        <p className="mt-2 text-sm">
          <Link href="/me/git-secret" className="underline">Manage git secret</Link>
          <span className="ml-2 text-xs text-zinc-500">— used as your <code>git push</code> password.</span>
        </p>
      </section>
    </main>
  );
}
