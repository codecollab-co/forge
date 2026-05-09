"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";

export default function UploadPage() {
  const router = useRouter();
  const params = useParams<{ owner: string; name: string }>();
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) router.replace("/auth");
    })();
  }, [router]);

  function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const list = Array.from(e.target.files ?? []);
    setFiles(list.filter((f) => !f.webkitRelativePath.includes("/.git/") && !f.webkitRelativePath.startsWith(".git/")));
  }

  async function onUpload(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await api.uploadFiles(params.owner, params.name, files);
      if (result.pr_number) {
        router.push(`/${params.owner}/${params.name}/pulls/${result.pr_number}`);
      } else {
        router.push(`/${params.owner}/${params.name}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-3xl space-y-6 px-6 py-8">
        <h1 className="text-2xl font-semibold">Upload files</h1>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Pick a folder. Files land on a new branch and a Pull Request opens
          against the default branch — review then merge. 50 MB cap.
        </p>

        <form onSubmit={onUpload} className="space-y-4">
          <input
            type="file"
            {...({ webkitdirectory: "", directory: "" } as Record<string, string>)}
            multiple
            onChange={onPick}
            className="block text-sm"
          />
          {files.length > 0 && (
            <p className="text-xs text-zinc-500">
              {files.length} files (
              {(files.reduce((s, f) => s + f.size, 0) / 1024 / 1024).toFixed(2)} MB)
            </p>
          )}
          {error && <p className="text-red-600">Error: {error}</p>}
          <button
            type="submit"
            disabled={busy || files.length === 0}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          >
            {busy ? "Uploading…" : "Upload and open PR"}
          </button>
        </form>
      </main>
    </>
  );
}
