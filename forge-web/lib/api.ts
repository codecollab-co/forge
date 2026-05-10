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

export type TokenSummary = {
  id: string;
  name: string;
  scopes: string[];
  created_at: string;
  expires_at: string | null;
  last_used_at: string | null;
};

export type TokenListResponse = {
  username: string;
  tokens: TokenSummary[];
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

export type Commit = {
  oid: string;
  short_oid: string;
  author_name: string;
  author_email: string;
  author_date: string;
  subject: string;
  parents: string[];
};

export type CommitDetail = Commit & { diff: string };

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
  updateMe: (patch: { display_name?: string; handle?: string }) =>
    request<Me>("PATCH", "/me", patch),
  listMyRuns: () => request<DashboardRun[]>("GET", "/me/runs"),

  listRepos: () => request<Repo[]>("GET", "/repos"),
  createRepo: (input: {
    name: string;
    description?: string;
    visibility?: "public" | "private";
    init_readme?: boolean;
    import_url?: string;
  }) => request<Repo>("POST", "/repos", input),
  updateRepo: (
    owner: string,
    name: string,
    patch: { description?: string; visibility?: "public" | "private"; name?: string },
  ) => request<Repo>("PATCH", `/repos/${owner}/${name}`, patch),
  deleteRepo: async (owner: string, name: string) => {
    const res = await fetch(`${apiDomain}/repos/${owner}/${name}`, {
      method: "DELETE",
      credentials: "include",
    });
    if (!res.ok && res.status !== 204) throw new Error(`DELETE → ${res.status}`);
  },
  uploadFiles: async (
    owner: string,
    name: string,
    files: File[],
    opts: { commit_subject?: string; commit_body?: string; commit_mode?: "direct" | "branch"; branch_name?: string } = {},
  ) => {
    const fd = new FormData();
    for (const f of files) {
      const path = (f as File & { webkitRelativePath?: string }).webkitRelativePath || f.name;
      fd.append(path, f, f.name);
    }
    if (opts.commit_subject) fd.append("commit_subject", opts.commit_subject);
    if (opts.commit_body) fd.append("commit_body", opts.commit_body);
    if (opts.commit_mode) fd.append("commit_mode", opts.commit_mode);
    if (opts.branch_name) fd.append("branch_name", opts.branch_name);
    const res = await fetch(`${apiDomain}/repos/${owner}/${name}/upload`, {
      method: "POST",
      credentials: "include",
      body: fd,
    });
    if (!res.ok) throw new Error(`upload → ${res.status}: ${await res.text()}`);
    return (await res.json()) as { branch: string; commit_oid: string; pr_number: number };
  },
  getRepo: (owner: string, name: string) => request<Repo>("GET", `/repos/${owner}/${name}`),
  getBranches: (owner: string, name: string) =>
    request<Branches>("GET", `/repos/${owner}/${name}/branches`),
  createBranch: (owner: string, name: string, body: { name: string; from?: string }) =>
    request<{ name: string; from: string }>("POST", `/repos/${owner}/${name}/branches`, body),
  renameBranch: (owner: string, name: string, branch: string, newName: string) =>
    request<{ new_name: string }>(
      "PATCH",
      `/repos/${owner}/${name}/branches/${encodeURIComponent(branch)}`,
      { new_name: newName },
    ),
  deleteBranch: async (owner: string, name: string, branch: string) => {
    const res = await fetch(
      `${apiDomain}/repos/${owner}/${name}/branches/${encodeURIComponent(branch)}`,
      { method: "DELETE", credentials: "include" },
    );
    if (!res.ok && res.status !== 204) throw new Error(`DELETE → ${res.status}: ${await res.text()}`);
  },

  listCommits: (owner: string, name: string, branch: string, limit = 50, offset = 0) =>
    request<Commit[]>(
      "GET",
      `/repos/${owner}/${name}/commits/${encodeURIComponent(branch)}?limit=${limit}&offset=${offset}`,
    ),
  getCommit: (owner: string, name: string, oid: string) =>
    request<CommitDetail>("GET", `/repos/${owner}/${name}/commit/${oid}`),
  getTree: (owner: string, name: string, ref: string, dir = "") =>
    request<TreeEntry[]>(
      "GET",
      `/repos/${owner}/${name}/tree/${encodeURIComponent(ref)}?path=${encodeURIComponent(dir)}`,
    ),

  listTokens: () => request<TokenListResponse>("GET", "/me/tokens"),
  mintToken: (name: string, expiresInDays?: number) =>
    request<{ username: string; token: TokenSummary; secret: string }>(
      "POST", "/me/tokens",
      { name, expires_in_days: expiresInDays ?? 0 },
    ),
  revokeToken: async (id: string) => {
    const res = await fetch(`${apiDomain}/me/tokens/${id}`, {
      method: "DELETE", credentials: "include",
    });
    if (!res.ok && res.status !== 204) throw new Error(`DELETE → ${res.status}`);
  },

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
  mergePull: (owner: string, name: string, number: number, opts: { delete_branch?: boolean } = {}) =>
    request<{ merge_commit_oid: string; state: string }>(
      "POST",
      `/repos/${owner}/${name}/pulls/${number}/merge`,
      opts,
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
