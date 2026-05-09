"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type Branches, type Me, type Repo } from "@/lib/api";
import { RepoTabs } from "@/app/components/RepoTabs";
import { UploadForm } from "@/app/components/UploadForm";

export default function UploadPage() {
  const params = useParams<{ owner: string; name: string }>();
  const [me, setMe] = useState<Me | null>(null);
  const [repo, setRepo] = useState<Repo | null>(null);
  const [branches, setBranches] = useState<Branches | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const [m, r, b] = await Promise.all([
          api.me().catch(() => null),
          api.getRepo(params.owner, params.name),
          api.getBranches(params.owner, params.name),
        ]);
        setMe(m);
        setRepo(r);
        setBranches(b);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [params.owner, params.name]);

  return (
    <>
      <RepoTabs owner={params.owner} name={params.name} />
      <main className="mx-auto max-w-3xl space-y-6 px-6 py-8">
        <h1 className="text-2xl font-semibold">Upload files</h1>
        {error && <p className="text-red-600">Error: {error}</p>}
        {repo && (
          <UploadForm
            owner={params.owner}
            name={params.name}
            defaultBranch={branches?.default || ""}
            isOwner={!!me && me.handle === repo.owner}
          />
        )}
      </main>
    </>
  );
}
