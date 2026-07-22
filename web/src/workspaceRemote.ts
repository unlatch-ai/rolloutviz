import { clearStoredDaemonToken, daemonToken } from "./api";
import type { WorkspaceState } from "./workspace";
import { normalizeWorkspace, workspaceWithoutLayout } from "./workspace";

export type RemoteWorkspace = {
  workspace_id: string;
  revision: number;
  workspace: WorkspaceState;
};

export function remoteWorkspaceID(search = globalThis.location?.search ?? ""): string | undefined {
  return new URLSearchParams(search).get("workspace_id") ?? undefined;
}

function headers(write = false): Record<string, string> {
  const result: Record<string, string> = { Accept: "application/json" };
  const token = daemonToken();
  if (token) result.Authorization = `Bearer ${token}`;
  if (write) result["Content-Type"] = "application/json";
  return result;
}

async function responseWorkspace(response: Response): Promise<RemoteWorkspace> {
  if (response.status === 401 || response.status === 403) clearStoredDaemonToken();
  if (!response.ok) throw new Error(`Workspace API returned ${response.status}`);
  const value = await response.json() as Partial<RemoteWorkspace>;
  const workspace = normalizeWorkspace(value.workspace);
  if (!workspace || typeof value.workspace_id !== "string" || typeof value.revision !== "number") throw new Error("Workspace API returned invalid state");
  return { workspace_id: value.workspace_id, revision: value.revision, workspace };
}

export async function loadRemoteWorkspace(id: string, after: number, signal?: AbortSignal): Promise<RemoteWorkspace> {
  const query = after >= 0 ? `?after=${after}` : "";
  const response = await fetch(`/api/v1/workspaces/${encodeURIComponent(id)}${query}`, { signal, headers: headers() });
  return responseWorkspace(response);
}

export async function saveRemoteWorkspace(id: string, workspace: WorkspaceState, signal?: AbortSignal): Promise<RemoteWorkspace> {
  const response = await fetch(`/api/v1/workspaces/${encodeURIComponent(id)}`, {
    method: "PUT",
    signal,
    headers: headers(true),
    body: JSON.stringify(workspaceWithoutLayout(workspace)),
  });
  return responseWorkspace(response);
}
