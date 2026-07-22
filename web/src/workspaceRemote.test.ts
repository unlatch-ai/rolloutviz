import { afterEach, describe, expect, it, vi } from "vitest";
import { loadRemoteWorkspace, remoteWorkspaceID, saveRemoteWorkspace } from "./workspaceRemote";
import { emptyWorkspace } from "./workspace";

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("remote workspace control", () => {
  it("reads the named workspace from the URL", () => {
    expect(remoteWorkspaceID("?workspace_id=research-1")).toBe("research-1");
    expect(remoteWorkspaceID("?workspace=%7B%7D")).toBeUndefined();
  });

  it("long-polls and saves bounded logical state with daemon authentication", async () => {
    localStorage.setItem("rlviz.daemon-token", "secret");
    const workspace = emptyWorkspace();
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ workspace_id: "one", revision: 3, workspace }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ workspace_id: "one", revision: 4, workspace }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const loaded = await loadRemoteWorkspace("one", 2);
    expect(loaded.revision).toBe(3);
    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/v1/workspaces/one?after=2", expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer secret" }) }));

    await saveRemoteWorkspace("one", { ...workspace, layout: { local: true } as never });
    const request = fetchMock.mock.calls[1][1] as RequestInit;
    expect(request.method).toBe("PUT");
    expect(JSON.parse(String(request.body))).not.toHaveProperty("layout");
  });
});
