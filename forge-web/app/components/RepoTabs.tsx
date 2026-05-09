"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const TABS = [
  { key: "code", label: "Code", path: "" },
  { key: "issues", label: "Issues", path: "/issues" },
  { key: "pulls", label: "Pull requests", path: "/pulls" },
  { key: "settings", label: "Settings", path: "/settings" },
];

export function RepoTabs({ owner, name }: { owner: string; name: string }) {
  const pathname = usePathname();
  const base = `/${owner}/${name}`;
  const active = TABS.find((t) =>
    t.path === ""
      ? pathname === base || pathname.startsWith(`${base}/blob`) || pathname === `${base}/`
      : pathname === `${base}${t.path}` || pathname.startsWith(`${base}${t.path}/`),
  )?.key ?? "code";

  return (
    <nav className="border-b border-zinc-200 dark:border-zinc-800">
      <div className="mx-auto flex max-w-5xl gap-1 px-6">
        {TABS.map((t) => {
          const isActive = t.key === active;
          return (
            <Link
              key={t.key}
              href={`${base}${t.path}`}
              className={
                "border-b-2 px-3 py-3 text-sm transition-colors " +
                (isActive
                  ? "border-zinc-900 font-medium text-zinc-900 dark:border-zinc-100 dark:text-zinc-100"
                  : "border-transparent text-zinc-600 hover:border-zinc-300 hover:text-zinc-900 dark:text-zinc-400 dark:hover:border-zinc-700 dark:hover:text-zinc-100")
              }
            >
              {t.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
