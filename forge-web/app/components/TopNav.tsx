"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Session, { signOut } from "supertokens-auth-react/recipe/session";
import { api, type Me } from "@/lib/api";

export function TopNav() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [newOpen, setNewOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    (async () => {
      try {
        if (await Session.doesSessionExist()) {
          setMe(await api.me());
        }
      } catch {
        /* unauthenticated or transient — leave me=null */
      } finally {
        setLoaded(true);
      }
    })();
  }, []);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
        setNewOpen(false);
      }
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  async function onSignOut() {
    await signOut();
    router.replace("/");
    router.refresh();
  }

  return (
    <header className="border-b border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
      <nav className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
        <Link href="/" className="text-lg font-semibold tracking-tight">
          Forge
        </Link>

        <div ref={menuRef} className="flex items-center gap-3 text-sm">
          {loaded && !me && (
            <Link
              href="/auth"
              className="rounded-md bg-zinc-900 px-3 py-1.5 text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900"
            >
              Sign in
            </Link>
          )}
          {me && (
            <>
              <div className="relative">
                <button
                  onClick={() => { setNewOpen((v) => !v); setMenuOpen(false); }}
                  className="rounded-md border border-zinc-300 px-3 py-1.5 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
                >
                  + New
                </button>
                {newOpen && (
                  <div className="absolute right-0 mt-1 w-48 rounded-md border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
                    <Link
                      href="/repos/new"
                      onClick={() => setNewOpen(false)}
                      className="block px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800"
                    >
                      New repository
                    </Link>
                  </div>
                )}
              </div>

              <div className="relative">
                <button
                  onClick={() => { setMenuOpen((v) => !v); setNewOpen(false); }}
                  className="flex items-center gap-2 rounded-full border border-zinc-300 p-0.5 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
                  aria-label="Open profile menu"
                >
                  {me.avatar_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={me.avatar_url} alt="" className="h-7 w-7 rounded-full" />
                  ) : (
                    <span className="grid h-7 w-7 place-items-center rounded-full bg-zinc-300 text-xs font-semibold dark:bg-zinc-700">
                      {me.handle.slice(0, 1).toUpperCase()}
                    </span>
                  )}
                </button>
                {menuOpen && (
                  <div className="absolute right-0 mt-1 w-56 rounded-md border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
                    <div className="border-b border-zinc-200 px-3 py-2 text-xs text-zinc-500 dark:border-zinc-800">
                      Signed in as <span className="font-medium text-zinc-900 dark:text-zinc-100">@{me.handle}</span>
                    </div>
                    <Link href="/me" onClick={() => setMenuOpen(false)} className="block px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                      Your profile
                    </Link>
                    <Link href="/repos" onClick={() => setMenuOpen(false)} className="block px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                      Your repositories
                    </Link>
                    <Link href="/me/tokens" onClick={() => setMenuOpen(false)} className="block px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                      Personal access tokens
                    </Link>
                    <Link href="/me/ssh-keys" onClick={() => setMenuOpen(false)} className="block px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                      SSH keys
                    </Link>
                    <button
                      onClick={onSignOut}
                      className="block w-full border-t border-zinc-200 px-3 py-2 text-left text-sm hover:bg-zinc-50 dark:border-zinc-800 dark:hover:bg-zinc-800"
                    >
                      Sign out
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </nav>
    </header>
  );
}
