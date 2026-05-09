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
      <p className="text-zinc-600 dark:text-zinc-400">
        AI-native Git host. Walking-skeleton build (slice 1).
      </p>
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
