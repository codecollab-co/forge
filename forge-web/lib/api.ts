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

export const api = {
  me: () => request<Me>("GET", "/me"),
  listRepos: () => request<Repo[]>("GET", "/repos"),
  createRepo: (input: { name: string; description?: string; visibility?: "public" | "private" }) =>
    request<Repo>("POST", "/repos", input),
  getRepo: (owner: string, name: string) => request<Repo>("GET", `/repos/${owner}/${name}`),
};
