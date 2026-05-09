type Props = { params: Promise<{ owner: string; name: string }> };

const PLATFORM_API_URL = process.env.PLATFORM_API_URL ?? "http://localhost:8080";
const PLATFORM_PUBLIC_URL =
  process.env.NEXT_PUBLIC_PLATFORM_API_URL ?? "http://localhost:8080";

type RepoDTO = {
  id: string;
  owner: string;
  name: string;
  description: string;
  visibility: "public" | "private";
  created_at: string;
  clone_url: string;
};

async function fetchRepo(owner: string, name: string): Promise<RepoDTO | null> {
  const res = await fetch(`${PLATFORM_API_URL}/repos/${owner}/${name}`, {
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return (await res.json()) as RepoDTO;
}

export default async function RepoPage({ params }: Props) {
  const { owner, name } = await params;
  const repo = await fetchRepo(owner, name);

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

  const cloneURL = `${PLATFORM_PUBLIC_URL}${repo.clone_url}`;

  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-16">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold">
          {repo.owner}/{repo.name}
        </h1>
        {repo.description && (
          <p className="text-zinc-600 dark:text-zinc-400">{repo.description}</p>
        )}
        <p className="text-xs uppercase tracking-wide text-zinc-400">{repo.visibility}</p>
      </header>

      <section className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
        <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-zinc-500">
          Clone
        </h2>
        <pre className="overflow-x-auto text-sm">git clone {cloneURL}</pre>
      </section>

      <p className="text-sm text-zinc-500">
        File browser lands in slice 4. Push lands in slice 4 as well — clone of
        an empty Repository works today.
      </p>
    </main>
  );
}
