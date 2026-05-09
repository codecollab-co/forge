import { apiDomain } from "@/lib/supertokens";

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`${apiDomain}${path}`, {
    method,
    credentials: "include",
    headers: body ? { "content-type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${method} ${path} → ${res.status}: ${text || res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export type Me = {
  id: string;
  handle: string;
  email: string;
  display_name: string;
  avatar_url: string;
  provider: string;
};

export type Repo = {
  id: string;
  owner: string;
  name: string;
  description: string;
  visibility: "public" | "private";
  created_at: string;
  clone_url: string;
};

export type TreeEntry = {
  path: string;
  type: "blob" | "tree";
  mode: string;
  oid: string;
};

export type Branches = {
  default: string;
  branches: string[] | null;
};

export type GitSecretInfo = {
  exists: boolean;
  created_at: string | null;
  last_used_at: string | null;
  username: string;
};

export const api = {
  me: () => request<Me>("GET", "/me"),
  listRepos: () => request<Repo[]>("GET", "/repos"),
  createRepo: (input: { name: string; description?: string; visibility?: "public" | "private" }) =>
    request<Repo>("POST", "/repos", input),
  getRepo: (owner: string, name: string) => request<Repo>("GET", `/repos/${owner}/${name}`),
  getBranches: (owner: string, name: string) =>
    request<Branches>("GET", `/repos/${owner}/${name}/branches`),
  getTree: (owner: string, name: string, ref: string, dir = "") =>
    request<TreeEntry[]>(
      "GET",
      `/repos/${owner}/${name}/tree/${encodeURIComponent(ref)}?path=${encodeURIComponent(dir)}`,
    ),
  getGitSecretInfo: () => request<GitSecretInfo>("GET", "/me/git-secret"),
  generateGitSecret: () =>
    request<{ username: string; secret: string }>("POST", "/me/git-secret"),
};
