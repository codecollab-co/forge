"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type Me, type Repo } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";

export default function RepoSettingsPage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string }>();
  const [me, setMe] = useState<Me | null>(null);
  const [repo, setRepo] = useState<Repo | null>(null);

  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [newName, setNewName] = useState("");

  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        const [m, r] = await Promise.all([api.me(), api.getRepo(params.owner, params.name)]);
        setMe(m);
        setRepo(r);
        setDescription(r.description);
        setVisibility(r.visibility);
        setNewName(r.name);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name, router]);

  if (!me || !repo) {
    return (
      <>
        <RepoTabs owner={params.owner} name={params.name} />
        <main className="mx-auto max-w-3xl px-6 py-8 text-zinc-500">Loading…</main>
      </>
    );
  }

  if (me.handle !== repo.owner) {
    return (
      <>
        <RepoTabs owner={params.owner} name={params.name} />
        <main className="mx-auto max-w-3xl px-6 py-8">
          <p className="text-zinc-600 dark:text-zinc-400">Only the repository owner can change settings.</p>
        </main>
      </>
    );
  }

  async function saveGeneral(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const patch: { description?: string; visibility?: "public" | "private"; name?: string } = {
        description,
        visibility,
      };
      const renaming = newName !== repo!.name;
      if (renaming) patch.name = newName;
      const updated = await api.updateRepo(params.owner, params.name, patch);
      setRepo(updated);
      setSavedAt(Date.now());
      if (renaming) router.replace(`/${updated.owner}/${updated.name}/settings`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function deleteRepo() {
    const confirmed = window.prompt(
      `This will permanently delete ${repo!.owner}/${repo!.name} and its history.\nType the repo name to confirm:`,
    );
    if (confirmed !== repo!.name) return;
    setBusy(true);
    setError(null);
    try {
      await api.deleteRepo(params.owner, params.name);
      router.replace("/repos");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-3xl space-y-8 px-6 py-8">
        <h1 className="text-2xl font-semibold">Settings</h1>

        {error && <p className="text-red-600">Error: {error}</p>}

        <section className="space-y-4 rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
          <h2 className="text-sm font-medium uppercase tracking-wide text-zinc-500">General</h2>
          <form onSubmit={saveGeneral} className="space-y-4">
            <div>
              <label className="block text-sm font-medium">Repository name</label>
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value.toLowerCase())}
                className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
              />
              {newName !== repo.name && (
                <p className="mt-1 text-xs text-amber-600">
                  Renaming will break existing clone URLs.
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium">Description</label>
              <input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Visibility</label>
              <select
                value={visibility}
                onChange={(e) => setVisibility(e.target.value as "public" | "private")}
                className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
              >
                <option value="public">Public</option>
                <option value="private">Private</option>
              </select>
            </div>
            <button
              type="submit"
              disabled={busy}
              className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
            >
              Save
            </button>
            {savedAt && <span className="ml-3 text-xs text-emerald-600">Saved.</span>}
          </form>
        </section>

        <section className="space-y-3 rounded-md border border-red-300 p-6 dark:border-red-900">
          <h2 className="text-sm font-medium uppercase tracking-wide text-red-700 dark:text-red-400">
            Danger zone
          </h2>
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            Deleting removes the repository, its history, and every issue and PR on it.
          </p>
          <button
            onClick={deleteRepo}
            disabled={busy}
            className="rounded-md border border-red-400 px-4 py-2 text-sm text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
          >
            Delete this repository
          </button>
        </section>
      </main>
    </>
  );
}
