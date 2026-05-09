"use client";

import { useEffect, useState } from "react";
import Session, { signOut } from "supertokens-auth-react/recipe/session";
import { useRouter } from "next/navigation";
import { apiDomain } from "@/lib/supertokens";

type Me = {
  id: string;
  handle: string;
  email: string;
  display_name: string;
  avatar_url: string;
  provider: string;
};

export default function MePage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      const hasSession = await Session.doesSessionExist();
      if (!hasSession) {
        router.replace("/auth");
        return;
      }
      try {
        const res = await fetch(`${apiDomain}/me`, { credentials: "include" });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        setMe((await res.json()) as Me);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [router]);

  async function onSignOut() {
    await signOut();
    router.replace("/");
  }

  return (
    <main className="mx-auto max-w-2xl space-y-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Profile</h1>
      {error && <p className="text-red-600">Error: {error}</p>}
      {me && (
        <section className="space-y-2 rounded-md border border-zinc-200 p-4 dark:border-zinc-800">
          {me.avatar_url && (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={me.avatar_url} alt="" className="h-16 w-16 rounded-full" />
          )}
          <p className="text-lg font-medium">{me.display_name || me.handle}</p>
          <p className="text-zinc-600 dark:text-zinc-400">@{me.handle}</p>
          <p className="text-sm text-zinc-500">{me.email}</p>
          <p className="text-xs uppercase tracking-wide text-zinc-400">via {me.provider}</p>
        </section>
      )}
      <div className="flex gap-3">
        <a
          href="/me/git-secret"
          className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Manage git secret
        </a>
        <button
          onClick={onSignOut}
          className="rounded-md border border-zinc-300 px-4 py-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Sign out
        </button>
      </div>
    </main>
  );
}
