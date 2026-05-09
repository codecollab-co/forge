import Link from "next/link";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { RepoTabs } from "@/app/components/RepoTabs";
import { CodeDropdown } from "@/app/components/CodeDropdown";
import { BranchSwitcher } from "@/app/components/BranchSwitcher";

type Props = {
  params: Promise<{ owner: string; name: string }>;
  searchParams: Promise<{ branch?: string }>;
};

const PLATFORM_API_URL = process.env.PLATFORM_API_URL ?? "http://localhost:8080";
const PLATFORM_PUBLIC_URL =
  process.env.NEXT_PUBLIC_PLATFORM_API_URL ?? "http://localhost:8080";

type RepoDTO = {
  id: string;
  owner: string;
  name: string;
  description: string;
  visibility: "public" | "private";
  clone_url: string;
};
type Branches = { default: string; branches: string[] | null };
type TreeEntry = { path: string; type: "blob" | "tree"; mode: string; oid: string };

async function get<T>(path: string): Promise<T | null> {
  const res = await fetch(`${PLATFORM_API_URL}${path}`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`HTTP ${res.status} on ${path}`);
  return (await res.json()) as T;
}

async function getText(path: string): Promise<string | null> {
  const res = await fetch(`${PLATFORM_API_URL}${path}`, { cache: "no-store" });
  if (res.status === 404 || !res.ok) return null;
  return res.text();
}

export default async function RepoPage({ params, searchParams }: Props) {
  const { owner, name } = await params;
  const { branch } = await searchParams;

  const repo = await get<RepoDTO>(`/repos/${owner}/${name}`);
  if (!repo) {
    return (
      <main className="mx-auto max-w-3xl px-6 py-16">
        <h1 className="text-2xl font-semibold">Not found</h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          {owner}/{name} does not exist.
        </p>
      </main>
    );
  }

  const branchesInfo = (await get<Branches>(`/repos/${owner}/${name}/branches`)) ?? {
    default: "",
    branches: [],
  };
  const ref = branch || branchesInfo.default || "main";
  const allBranches = branchesInfo.branches ?? [];
  const isEmpty = allBranches.length === 0;

  const tree = isEmpty ? [] : (await get<TreeEntry[]>(`/repos/${owner}/${name}/tree/${ref}`)) ?? [];
  const readmeEntry = tree.find((e) => e.type === "blob" && e.path.toLowerCase() === "readme.md");
  const readme = readmeEntry
    ? await getText(`/repos/${owner}/${name}/blob/${ref}?path=readme.md`)
    : null;

  const cloneURL = `${PLATFORM_PUBLIC_URL}${repo.clone_url}`;

  return (
    <>
      <RepoTabs owner={owner} name={name} />
      <main className="mx-auto max-w-5xl space-y-6 px-6 py-8">
        <header className="space-y-1">
          <h1 className="text-2xl font-semibold">
            {repo.owner}/{repo.name}
          </h1>
          {repo.description && (
            <p className="text-zinc-600 dark:text-zinc-400">{repo.description}</p>
          )}
          <p className="text-xs uppercase tracking-wide text-zinc-400">{repo.visibility}</p>
        </header>

        <div className="flex flex-wrap items-center justify-between gap-2">
          {!isEmpty ? (
            <BranchSwitcher
              owner={owner}
              name={name}
              current={ref}
              branches={allBranches}
              defaultBranch={branchesInfo.default || "main"}
              canCreate={true}
            />
          ) : (
            <span className="text-sm text-zinc-500">No branches yet.</span>
          )}
          <div className="flex items-center gap-2">
            <Link
              href={`/${owner}/${name}/upload`}
              className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
            >
              Upload files
            </Link>
            <CodeDropdown cloneURL={cloneURL} />
          </div>
        </div>

      {isEmpty ? (
        <section className="rounded-md border border-zinc-200 p-6 dark:border-zinc-800">
          <h2 className="text-base font-medium">Quick start</h2>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            This repository is empty. Push your first commit, or upload files
            to get started.
          </p>
          <pre className="mt-4 overflow-x-auto rounded-md border border-zinc-200 bg-zinc-50 p-3 text-xs dark:border-zinc-800 dark:bg-zinc-950">
            git clone {cloneURL}
          </pre>
        </section>
      ) : (
        <section className="rounded-md border border-zinc-200 dark:border-zinc-800">
          <h2 className="border-b border-zinc-200 px-4 py-2 text-sm font-medium uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
            Files on {ref}
          </h2>
          <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {tree.map((e) => (
              <li key={e.path} className="px-4 py-2 text-sm">
                {e.type === "blob" ? (
                  <Link
                    href={`/${owner}/${name}/blob/${encodeURIComponent(ref)}/${e.path}`}
                    className="hover:underline"
                  >
                    {e.path}
                  </Link>
                ) : (
                  <span className="font-medium">{e.path}/</span>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      {readme && (
          <section className="prose prose-zinc max-w-none rounded-md border border-zinc-200 p-6 dark:prose-invert dark:border-zinc-800">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{readme}</ReactMarkdown>
          </section>
        )}
      </main>
    </>
  );
}
