"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type Me } from "@/lib/api";

const NAME_RE = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$/;
const DESC_LIMIT = 350;

type Mode = "blank" | "readme" | "upload" | "import";
type CatalogItem = { key: string; name: string };

export default function NewRepoPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);

  // Step 1: General
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  // Step 2: Configure
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [mode, setMode] = useState<Mode>("readme");
  const [importURL, setImportURL] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [licenseKey, setLicenseKey] = useState("");
  const [gitignoreKey, setGitignoreKey] = useState("");

  // Catalogs
  const [licenses, setLicenses] = useState<CatalogItem[]>([]);
  const [gitignores, setGitignores] = useState<CatalogItem[]>([]);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        const [m, ls, gs] = await Promise.all([
          api.me(),
          api.listLicenses(),
          api.listGitignores(),
        ]);
        setMe(m);
        setLicenses(ls);
        setGitignores(gs);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  function onPickFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const list = Array.from(e.target.files ?? []);
    setFiles(
      list.filter(
        (f) =>
          !((f as File & { webkitRelativePath?: string }).webkitRelativePath ?? "").startsWith(".git/"),
      ),
    );
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!NAME_RE.test(name)) {
      setError("Name must be lowercase, alphanumeric + dash, 1–40 chars.");
      return;
    }
    if (description.length > DESC_LIMIT) {
      setError(`Description must be ${DESC_LIMIT} characters or fewer.`);
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
        license: mode === "import" ? undefined : licenseKey || undefined,
        gitignore: mode === "import" ? undefined : gitignoreKey || undefined,
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

  const ownerHandle = me?.handle ?? "you";

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-10">
      <h1 className="text-2xl font-semibold">Create a new repository</h1>

      <form onSubmit={onSubmit} className="space-y-8">
        <fieldset className="space-y-4 rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
          <legend className="px-2 text-sm font-medium uppercase tracking-wide text-zinc-500">
            General
          </legend>

          <div className="grid grid-cols-[auto_auto_1fr] items-end gap-2">
            <div>
              <label className="block text-sm font-medium">Owner</label>
              <select
                value={ownerHandle}
                disabled
                className="mt-1 rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900"
              >
                <option>{ownerHandle}</option>
              </select>
            </div>
            <span className="px-1 pb-2 text-lg text-zinc-500">/</span>
            <div>
              <label className="block text-sm font-medium">
                Repository name <span className="text-red-600">*</span>
              </label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value.toLowerCase())}
                className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
                placeholder="hello-world"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value.slice(0, DESC_LIMIT))}
              rows={3}
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
            />
            <p className="mt-1 text-xs text-zinc-500">
              {description.length} / {DESC_LIMIT}
            </p>
          </div>
        </fieldset>

        <fieldset className="space-y-5 rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
          <legend className="px-2 text-sm font-medium uppercase tracking-wide text-zinc-500">
            Configure
          </legend>

          <div>
            <p className="text-sm font-medium">
              Visibility <span className="text-red-600">*</span>
            </p>
            <div className="mt-2 space-y-2">
              <label className="flex items-start gap-2 text-sm">
                <input
                  type="radio"
                  name="visibility"
                  checked={visibility === "public"}
                  onChange={() => setVisibility("public")}
                  className="mt-1"
                />
                <span>
                  <span className="font-medium">Public</span>
                  <span className="block text-xs text-zinc-500">
                    Anyone on the internet can see this repository. You choose who can commit.
                  </span>
                </span>
              </label>
              <label className="flex items-start gap-2 text-sm">
                <input
                  type="radio"
                  name="visibility"
                  checked={visibility === "private"}
                  onChange={() => setVisibility("private")}
                  className="mt-1"
                />
                <span>
                  <span className="font-medium">Private</span>
                  <span className="block text-xs text-zinc-500">
                    You choose who can see and commit to this repository.
                  </span>
                </span>
              </label>
            </div>
          </div>

          <div>
            <p className="text-sm font-medium">Initial contents</p>
            <div className="mt-2 space-y-2">
              <ModeRadio mode="readme" current={mode} onChange={setMode} label="Initialize with a README">
                Creates the first commit on <code>main</code> from the templates below.
              </ModeRadio>
              <ModeRadio mode="blank" current={mode} onChange={setMode} label="Empty repository">
                No initial commit.
              </ModeRadio>
              <ModeRadio mode="upload" current={mode} onChange={setMode} label="Upload a folder">
                Up to 50 MB, single commit, no git history preserved.
              </ModeRadio>
              {mode === "upload" && (
                <div className="ml-6">
                  <input
                    ref={fileInputRef}
                    type="file"
                    {...({ webkitdirectory: "", directory: "" } as Record<string, string>)}
                    multiple
                    onChange={onPickFiles}
                    className="block text-sm"
                  />
                  {files.length > 0 && (
                    <p className="mt-1 text-xs text-zinc-500">
                      {files.length} file{files.length === 1 ? "" : "s"} (
                      {(files.reduce((s, f) => s + f.size, 0) / 1024 / 1024).toFixed(2)} MB)
                    </p>
                  )}
                </div>
              )}
              <ModeRadio mode="import" current={mode} onChange={setMode} label="Import from a Git URL">
                Server-side mirror clone. Public URLs only at MVP.
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
            </div>
          </div>

          {mode !== "import" && (
            <>
              <div>
                <label className="block text-sm font-medium">Add .gitignore</label>
                <select
                  value={gitignoreKey}
                  onChange={(e) => setGitignoreKey(e.target.value)}
                  className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
                >
                  <option value="">No .gitignore</option>
                  {gitignores.map((g) => (
                    <option key={g.key} value={g.key}>{g.name}</option>
                  ))}
                </select>
                <p className="mt-1 text-xs text-zinc-500">
                  <code>.gitignore</code> tells git which files to ignore.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium">License</label>
                <select
                  value={licenseKey}
                  onChange={(e) => setLicenseKey(e.target.value)}
                  className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
                >
                  <option value="">No license</option>
                  {licenses.map((l) => (
                    <option key={l.key} value={l.key}>{l.name}</option>
                  ))}
                </select>
                <p className="mt-1 text-xs text-zinc-500">
                  Tells others how they can use your code.
                </p>
              </div>
            </>
          )}
        </fieldset>

        {error && <p className="text-red-600">Error: {error}</p>}

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-700 disabled:opacity-50"
          >
            {submitting ? "Creating…" : "Create repository"}
          </button>
          <button
            type="button"
            onClick={() => router.push("/repos")}
            className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
          >
            Cancel
          </button>
        </div>
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
