"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api } from "@/lib/api";

const NAME_RE = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$/;

export default function NewRepoPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [initReadme, setInitReadme] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) router.replace("/auth");
    })();
  }, [router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!NAME_RE.test(name)) {
      setError("Name must be lowercase, alphanumeric + dash, 1–40 chars.");
      return;
    }
    setSubmitting(true);
    try {
      const repo = await api.createRepo({ name, description, visibility, init_readme: initReadme });
      router.push(`/${repo.owner}/${repo.name}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setSubmitting(false);
    }
  }

  return (
    <main className="mx-auto max-w-xl space-y-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">New Repository</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium">Name</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value.toLowerCase())}
            className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
            placeholder="hello-world"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium">Description (optional)</label>
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
        <label className="flex items-start gap-2 text-sm">
          <input
            type="checkbox"
            checked={initReadme}
            onChange={(e) => setInitReadme(e.target.checked)}
            className="mt-1"
          />
          <span>
            <span className="font-medium">Initialize with a README.md</span>
            <span className="block text-xs text-zinc-500">
              Creates the first commit on <code>main</code> so you can clone and edit immediately.
              License picker and <code>.gitignore</code> templates land in a follow-up.
            </span>
          </span>
        </label>

        {error && <p className="text-red-600">Error: {error}</p>}
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        >
          {submitting ? "Creating…" : "Create Repository"}
        </button>
      </form>
    </main>
  );
}
