"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api } from "@/lib/api";

const NAME_RE = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$/;

type Mode = "blank" | "readme" | "upload" | "import";

export default function NewRepoPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [mode, setMode] = useState<Mode>("readme");
  const [importURL, setImportURL] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) router.replace("/auth");
    })();
  }, [router]);

  function onPickFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const list = Array.from(e.target.files ?? []);
    setFiles(list.filter((f) => !f.webkitRelativePath.includes("/.git/") && !f.webkitRelativePath.startsWith(".git/")));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!NAME_RE.test(name)) {
      setError("Name must be lowercase, alphanumeric + dash, 1–40 chars.");
      return;
    }
    if (mode === "import" && !importURL.trim()) {
      setError("Import requires a Git URL.");
      return;
    }
    if (mode === "upload" && files.length === 0) {
      setError("Pick a folder to upload, or switch mode.");
      return;
    }
    setSubmitting(true);
    try {
      const repo = await api.createRepo({
        name,
        description,
        visibility,
        init_readme: mode === "readme",
        import_url: mode === "import" ? importURL.trim() : undefined,
      });
      if (mode === "upload") {
        await api.uploadFiles(repo.owner, repo.name, files);
      }
      router.push(`/${repo.owner}/${repo.name}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setSubmitting(false);
    }
  }

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-10">
      <h1 className="text-2xl font-semibold">New Repository</h1>

      <form onSubmit={onSubmit} className="space-y-5">
        <div className="grid gap-4 sm:grid-cols-2">
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
        </div>

        <div>
          <label className="block text-sm font-medium">Description (optional)</label>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 dark:border-zinc-700 dark:bg-zinc-900"
          />
        </div>

        <fieldset className="space-y-3 rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
          <legend className="px-2 text-sm font-medium">Initial contents</legend>

          <ModeRadio mode="readme" current={mode} onChange={setMode} label="Initialize with a README">
            Creates the first commit on <code>main</code>. Good for a brand-new project.
          </ModeRadio>
          <ModeRadio mode="blank" current={mode} onChange={setMode} label="Empty repository">
            No initial commit. You'll see clone instructions on the next page.
          </ModeRadio>
          <ModeRadio mode="upload" current={mode} onChange={setMode} label="Upload a folder">
            Pick a folder from your machine. Up to 50 MB. Single commit, no
            git history is preserved.
          </ModeRadio>
          {mode === "upload" && (
            <div className="ml-6 space-y-2">
              <input
                ref={fileInputRef}
                type="file"
                {...({ webkitdirectory: "", directory: "" } as Record<string, string>)}
                multiple
                onChange={onPickFiles}
                className="block text-sm"
              />
              {files.length > 0 && (
                <p className="text-xs text-zinc-500">
                  {files.length} files (
                  {(files.reduce((s, f) => s + f.size, 0) / 1024 / 1024).toFixed(2)} MB)
                </p>
              )}
            </div>
          )}
          <ModeRadio mode="import" current={mode} onChange={setMode} label="Import from a Git URL">
            Server-side mirror clone. Preserves history. Public URLs only at MVP.
          </ModeRadio>
          {mode === "import" && (
            <div className="ml-6">
              <input
                value={importURL}
                onChange={(e) => setImportURL(e.target.value)}
                placeholder="https://github.com/user/repo.git"
                className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
              />
            </div>
          )}
        </fieldset>

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

function ModeRadio({
  mode,
  current,
  onChange,
  label,
  children,
}: {
  mode: Mode;
  current: Mode;
  onChange: (m: Mode) => void;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex items-start gap-2 text-sm">
      <input
        type="radio"
        name="mode"
        checked={current === mode}
        onChange={() => onChange(mode)}
        className="mt-1"
      />
      <span>
        <span className="font-medium">{label}</span>
        <span className="block text-xs text-zinc-500">{children}</span>
      </span>
    </label>
  );
}
