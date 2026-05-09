import Link from "next/link";

type Props = {
  params: Promise<{ owner: string; name: string; ref: string; path: string[] }>;
};

const PLATFORM_API_URL = process.env.PLATFORM_API_URL ?? "http://localhost:8080";

async function getText(path: string): Promise<string | null> {
  const res = await fetch(`${PLATFORM_API_URL}${path}`, { cache: "no-store" });
  if (res.status === 404 || !res.ok) return null;
  return res.text();
}

export default async function BlobPage({ params }: Props) {
  const { owner, name, ref, path } = await params;
  const filePath = path.join("/");
  const blob = await getText(
    `/repos/${owner}/${name}/blob/${encodeURIComponent(ref)}?path=${encodeURIComponent(filePath)}`,
  );

  return (
    <main className="mx-auto max-w-4xl space-y-4 px-6 py-12">
      <nav className="text-sm">
        <Link href={`/${owner}/${name}?branch=${encodeURIComponent(ref)}`} className="hover:underline">
          ← {owner}/{name}
        </Link>
      </nav>
      <h1 className="font-mono text-lg">{filePath}</h1>
      {blob === null ? (
        <p className="text-zinc-500">Not found, or binary content.</p>
      ) : (
        <pre className="overflow-x-auto rounded-md border border-zinc-200 bg-zinc-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-900">
          {blob}
        </pre>
      )}
    </main>
  );
}
