import Link from "next/link";

const PLATFORM_API_URL = process.env.PLATFORM_API_URL ?? "http://localhost:8080";

async function platformHealth(): Promise<{ ok: boolean; body: string }> {
  try {
    const res = await fetch(`${PLATFORM_API_URL}/healthz`, { cache: "no-store" });
    return { ok: res.ok, body: await res.text() };
  } catch (err) {
    return { ok: false, body: err instanceof Error ? err.message : String(err) };
  }
}

export default async function Page() {
  const { ok, body } = await platformHealth();
  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-16">
      <h1 className="text-3xl font-semibold">Forge</h1>
      <p className="text-zinc-600 dark:text-zinc-400">AI-native Git host.</p>

      <div className="flex gap-3">
        <Link
          href="/auth"
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          Sign in
        </Link>
        <Link
          href="/me"
          className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Profile
        </Link>
      </div>

      <section className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
        <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-zinc-500">
          forge-platform /healthz
        </h2>
        <pre className="overflow-x-auto text-sm">
          {ok ? body : `unreachable: ${body}`}
        </pre>
      </section>
    </main>
  );
}
