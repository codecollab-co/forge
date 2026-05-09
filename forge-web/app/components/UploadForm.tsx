"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type Me } from "@/lib/api";

type Props = {
  owner: string;
  name: string;
  defaultBranch: string;       // e.g. "main"; "" if repo is empty
  isOwner: boolean;
};

export function UploadForm({ owner, name, defaultBranch, isOwner }: Props) {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [files, setFiles] = useState<File[]>([]);
  const [dragOver, setDragOver] = useState(false);
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [mode, setMode] = useState<"direct" | "branch">(isOwner ? "direct" : "branch");
  const [branchName, setBranchName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        const m = await api.me();
        setMe(m);
        setBranchName(`${m.handle}-patch-1`);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  function pushFiles(list: FileList | File[]) {
    const arr = Array.from(list).filter(
      (f) => !((f as File & { webkitRelativePath?: string }).webkitRelativePath ?? "").startsWith(".git/"),
    );
    setFiles((prev) => [...prev, ...arr]);
  }

  function onDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    if (e.dataTransfer.files?.length) pushFiles(e.dataTransfer.files);
  }

  function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    if (e.target.files) pushFiles(e.target.files);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (files.length === 0) {
      setError("Pick or drop at least one file.");
      return;
    }
    setBusy(true);
    try {
      const result = await api.uploadFiles(owner, name, files, {
        commit_subject: subject.trim() || undefined,
        commit_body: body || undefined,
        commit_mode: defaultBranch ? mode : "direct",
        branch_name: mode === "branch" ? branchName.trim() : undefined,
      });
      if (result.pr_number) {
        router.push(`/${owner}/${name}/pulls/${result.pr_number}`);
      } else {
        router.push(`/${owner}/${name}`);
      }
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  const totalMB = files.reduce((s, f) => s + f.size, 0) / 1024 / 1024;
  const showRadios = !!defaultBranch; // hide entirely on a brand-new repo

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      <div
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
        className={
          "rounded-lg border-2 border-dashed p-8 text-center transition-colors " +
          (dragOver
            ? "border-zinc-900 bg-zinc-50 dark:border-zinc-100 dark:bg-zinc-900"
            : "border-zinc-300 dark:border-zinc-700")
        }
      >
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Drag files here to add them to your repository
        </p>
        <p className="mt-1 text-xs text-zinc-500">or</p>
        <div className="mt-3 flex justify-center gap-2">
          <label className="cursor-pointer rounded-md border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900">
            Choose files
            <input type="file" multiple onChange={onPick} className="hidden" />
          </label>
          <label className="cursor-pointer rounded-md border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900">
            Choose folder
            <input
              type="file"
              {...({ webkitdirectory: "", directory: "" } as Record<string, string>)}
              multiple
              onChange={onPick}
              className="hidden"
            />
          </label>
        </div>
        {files.length > 0 && (
          <p className="mt-3 text-xs text-zinc-500">
            {files.length} file{files.length === 1 ? "" : "s"} ({totalMB.toFixed(2)} MB) staged
          </p>
        )}
      </div>

      <fieldset className="space-y-3 rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
        <legend className="px-2 text-sm font-medium">Commit changes</legend>
        <div>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder={`Add ${files.length || "N"} file${files.length === 1 ? "" : "s"}`}
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
          />
          <p className="mt-1 text-xs text-zinc-500">Commit message (short).</p>
        </div>
        <div>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            placeholder="Add a more detailed description (optional)…"
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
          />
          <p className="mt-1 text-xs text-zinc-500">Extended description (optional).</p>
        </div>

        {showRadios && (
          <>
            <label className={`flex items-start gap-2 text-sm ${isOwner ? "" : "opacity-50"}`}>
              <input
                type="radio"
                name="mode"
                checked={mode === "direct"}
                onChange={() => setMode("direct")}
                disabled={!isOwner}
                className="mt-1"
              />
              <span>
                <span className="font-medium">Commit directly to the <code>{defaultBranch}</code> branch</span>
                {!isOwner && (
                  <span className="block text-xs text-zinc-500">
                    Only the repository owner can commit directly to {defaultBranch}.
                  </span>
                )}
              </span>
            </label>

            <label className="flex items-start gap-2 text-sm">
              <input
                type="radio"
                name="mode"
                checked={mode === "branch"}
                onChange={() => setMode("branch")}
                className="mt-1"
              />
              <span className="flex-1">
                <span className="font-medium">Create a new branch for this commit and start a pull request</span>
                {mode === "branch" && (
                  <input
                    value={branchName}
                    onChange={(e) => setBranchName(e.target.value)}
                    className="mt-2 w-full rounded-md border border-zinc-300 px-3 py-1.5 text-sm dark:border-zinc-700 dark:bg-zinc-900"
                    placeholder={me ? `${me.handle}-patch-1` : "branch-name"}
                  />
                )}
              </span>
            </label>
          </>
        )}
      </fieldset>

      {error && <p className="text-red-600">Error: {error}</p>}

      <div className="flex gap-3">
        <button
          type="submit"
          disabled={busy || files.length === 0}
          className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          {busy ? "Committing…" : "Commit changes"}
        </button>
        <button
          type="button"
          onClick={() => router.push(`/${owner}/${name}`)}
          className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
