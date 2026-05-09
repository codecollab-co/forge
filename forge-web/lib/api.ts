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

export type PRState = "open" | "merged" | "closed";

export type PullRequest = {
  id: string;
  number: number;
  title: string;
  body: string;
  head_branch: string;
  base_branch: string;
  state: PRState;
  author: string;
  merge_commit_oid: string;
  merged_at: string | null;
  created_at: string;
};

export type PullComment = {
  id: string;
  body: string;
  author: string;
  author_kind: "user" | "agent";
  created_at: string;
};

export type PullDetail = {
  pull_request: PullRequest;
  diff: string;
  comments: PullComment[];
};

export type IssueState = "open" | "closed";

export type Issue = {
  id: string;
  number: number;
  title: string;
  body: string;
  state: IssueState;
  author: string;
  assignee: { kind: "user" | "agent"; id: string; handle: string } | null;
  created_at: string;
  closed_at: string | null;
};

export type IssueComment = {
  id: string;
  body: string;
  author: string;
  created_at: string;
};

export type IssueDetail = {
  issue: Issue;
  comments: IssueComment[];
};

export type DashboardRun = {
  id: string;
  state: "queued" | "running" | "succeeded" | "failed" | "cancelled";
  issue_number: number;
  issue_title: string;
  repo_owner: string;
  repo_name: string;
  pr_number: number | null;
  created_at: string;
};

export const api = {
  me: () => request<Me>("GET", "/me"),
  updateMe: (patch: { display_name?: string }) =>
    request<Me>("PATCH", "/me", patch),
  listMyRuns: () => request<DashboardRun[]>("GET", "/me/runs"),

  listRepos: () => request<Repo[]>("GET", "/repos"),
  createRepo: (input: { name: string; description?: string; visibility?: "public" | "private"; init_readme?: boolean }) =>
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

  listPulls: (owner: string, name: string, state?: PRState) =>
    request<PullRequest[]>(
      "GET",
      `/repos/${owner}/${name}/pulls${state ? `?state=${state}` : ""}`,
    ),
  createPull: (owner: string, name: string, input: { title: string; body?: string; head_branch: string; base_branch: string }) =>
    request<PullRequest>("POST", `/repos/${owner}/${name}/pulls`, input),
  getPull: (owner: string, name: string, number: number) =>
    request<PullDetail>("GET", `/repos/${owner}/${name}/pulls/${number}`),
  addPullComment: (owner: string, name: string, number: number, body: string) =>
    request<PullComment>("POST", `/repos/${owner}/${name}/pulls/${number}/comments`, { body }),
  mergePull: (owner: string, name: string, number: number) =>
    request<{ merge_commit_oid: string; state: string }>(
      "POST",
      `/repos/${owner}/${name}/pulls/${number}/merge`,
    ),

  listIssues: (owner: string, name: string, state?: IssueState) =>
    request<Issue[]>(
      "GET",
      `/repos/${owner}/${name}/issues${state ? `?state=${state}` : ""}`,
    ),
  createIssue: (owner: string, name: string, input: { title: string; body?: string; assignee_user_id?: string }) =>
    request<Issue>("POST", `/repos/${owner}/${name}/issues`, input),
  getIssue: (owner: string, name: string, number: number) =>
    request<IssueDetail>("GET", `/repos/${owner}/${name}/issues/${number}`),
  addIssueComment: (owner: string, name: string, number: number, body: string) =>
    request<IssueComment>("POST", `/repos/${owner}/${name}/issues/${number}/comments`, { body }),
  closeIssue: (owner: string, name: string, number: number) =>
    request<Issue>("POST", `/repos/${owner}/${name}/issues/${number}/close`),
  reopenIssue: (owner: string, name: string, number: number) =>
    request<Issue>("POST", `/repos/${owner}/${name}/issues/${number}/reopen`),

  assignAgent: (owner: string, name: string, number: number) =>
    request<Run>("POST", `/repos/${owner}/${name}/issues/${number}/assign-agent`),
  getRun: (id: string) => request<Run>("GET", `/runs/${id}`),
  cancelRun: (id: string) =>
    request<{ cancel_requested: boolean }>("POST", `/runs/${id}/cancel`),
};

export type RunState = "queued" | "running" | "succeeded" | "failed" | "cancelled";

export type Run = {
  id: string;
  state: RunState;
  cancel_requested: boolean;
  sandbox_id: string;
  error_category: string;
  error_message: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
  last_heartbeat_at: string | null;
  pr_number: string;
};
