"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Session from "supertokens-auth-react/recipe/session";
import { api, type GitSecretInfo } from "@/lib/api";
import { CopyButton } from "@/app/components/CopyButton";

export default function GitSecretPage() {
  const router = useRouter();
  const [info, setInfo] = useState<GitSecretInfo | null>(null);
  const [revealed, setRevealed] = useState<{ username: string; secret: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    (async () => {
      if (!(await Session.doesSessionExist())) {
        router.replace("/auth");
        return;
      }
      try {
        setInfo(await api.getGitSecretInfo());
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  async function generate() {
    if (info?.exists && !confirm("Replace the existing secret? Existing git clients using it will stop working.")) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const r = await api.generateGitSecret();
      setRevealed(r);
      setInfo(await api.getGitSecretInfo());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Git secret</h1>
      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        Used as your <code>git push</code> password over HTTPS. Slice 12 will
        replace this with named, revocable Personal Access Tokens.
      </p>

      {error && <p className="text-red-600">Error: {error}</p>}

      {info && (
        <section className="rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
          <p className="text-sm">
            <span className="font-medium">Username:</span> <code>{info.username}</code>
          </p>
          {info.exists ? (
            <>
              <p className="mt-1 text-sm text-zinc-500">
                Created {info.created_at ? new Date(info.created_at).toLocaleString() : ""}
                {info.last_used_at &&
                  ` · last used ${new Date(info.last_used_at).toLocaleString()}`}
              </p>
            </>
          ) : (
            <p className="mt-1 text-sm text-zinc-500">No secret yet.</p>
          )}
        </section>
      )}

      <button
        onClick={generate}
        disabled={busy}
        className="rounded-md bg-zinc-900 px-4 py-2 text-sm text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
      >
        {info?.exists ? "Regenerate" : "Generate"}
      </button>

      {revealed && (
        <section className="space-y-3 rounded-md border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950">
          <p className="text-sm font-medium">
            Copy your secret now — it won't be shown again.
          </p>
          <div className="space-y-1">
            <p className="text-xs uppercase tracking-wide text-amber-900 dark:text-amber-200">
              Your git secret
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 overflow-x-auto rounded-md border border-amber-200 bg-white px-3 py-2 font-mono text-sm dark:border-amber-800 dark:bg-zinc-900">
                {revealed.secret}
              </code>
              <CopyButton text={revealed.secret} label="Copy secret" />
            </div>
          </div>
          <details className="text-xs text-zinc-600 dark:text-zinc-400">
            <summary className="cursor-pointer">How do I use this?</summary>
            <p className="mt-2">
              Use it as the password when git prompts on push/clone. Username
              is <code>{revealed.username}</code>. Example one-shot URL:
            </p>
            <pre className="mt-1 overflow-x-auto rounded-md border border-amber-200 bg-white px-3 py-2 dark:border-amber-800 dark:bg-zinc-900">
              git clone https://{revealed.username}:{revealed.secret}@localhost:8080/{revealed.username}/&lt;repo&gt;.git
            </pre>
            <p className="mt-2">
              Or set <code>git config credential.helper store</code> and let
              git cache it on first push.
            </p>
          </details>
        </section>
      )}
    </main>
  );
}
